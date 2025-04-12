// Package handler holds the handler which does all heavy lifting
package handler

import (
	"fmt"
	"os"
	"time"

	"github.com/pgvillage-tools/pgfga/internal/config"
	"github.com/pgvillage-tools/pgfga/pkg/ldap"
	"github.com/pgvillage-tools/pgfga/pkg/pg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log  *zap.SugaredLogger
	atom zap.AtomicLevel
)

// Initialize can be used to initialize this module with the logger
func Initialize() {
	atom = zap.NewAtomicLevel()
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atom,
	)).Sugar()

	pg.Initialize(log)
	ldap.Initialize(log)
}

// PgFgaHandler is a struct to hold the data that Handle uses.
// There is only one externally available Method (Handle) which will do all the heavy lifting.
// Handle stores all of his data in this struct.
type PgFgaHandler struct {
	config config.FgaConfig
	pg     *pg.Handler
	ldap   *ldap.Handler
}

// NewPgFgaHandler can be used to initialize an new Handler struct before calling Handle on it.
func NewPgFgaHandler() (pfh *PgFgaHandler, err error) {
	cnf, err := config.NewConfig()
	if err != nil {
		return pfh, err
	}

	atom.SetLevel(cnf.GeneralConfig.LogLevel)

	pfh = &PgFgaHandler{
		config: cnf,
	}

	pfh.ldap = ldap.NewLdapHandler(cnf.LdapConfig)

	pfh.pg = pg.NewPgHandler(cnf.PgDsn, cnf.StrictConfig, cnf.DbsConfig, cnf.Slots)

	return pfh, nil
}

// Handle will do all the heavy lifting of handling a PgFga run
func (pfh PgFgaHandler) Handle() {
	time.Sleep(pfh.config.GeneralConfig.RunDelay)

	for _, subHandler := range []func() error{
		pfh.handleRoles,
		pfh.handleUsers,
		pfh.handleDatabases,
		pfh.handleSlots,
	} {
		err := subHandler()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (pfh PgFgaHandler) handleLdapGroup(
	userConfig config.FgaUserConfig,
	userName string,
	options pg.RoleOptionMap,
) (err error) {
	log.Debugf("Configuring role from ldap for %s", userName)
	if userConfig.BaseDN == "" || userConfig.Filter == "" {
		return fmt.Errorf("ldapbasedn and ldapfilter must be set for %s (auth: 'ldap-group')", userName)
	}
	baseGroup, err := pfh.ldap.GetMembers(userConfig.BaseDN, userConfig.Filter)
	if err != nil {
		return err
	}
	baseRole, err := pg.NewRole(pfh.pg, baseGroup.Name(), options, userConfig.State)
	if err != nil {
		return err
	}
	err = baseRole.ResetPassword()
	if err != nil {
		return err
	}
	for _, ms := range baseGroup.MembershipTree() {
		_, err = pg.NewRole(pfh.pg, ms.GetMember().Name(), pg.RoleOptionMap{pg.RoleLogin: true}, userConfig.State)
		if err != nil {
			return err
		}
		err = pfh.pg.GrantRole(ms.GetMember().Name(), baseGroup.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func (pfh PgFgaHandler) handleLdapUser(
	userConfig config.FgaUserConfig,
	userName string,
	options *pg.RoleOptionMap,
) (err error) {
	log.Debugf("Configuring user %s with %s", userName, userConfig.Auth)
	options.AddAbsolute(pg.RoleLogin)
	user, err := pg.NewRole(pfh.pg, userName, *options, userConfig.State)
	if err != nil {
		return err
	}
	err = user.ResetPassword()
	if err != nil {
		return err
	}
	if userConfig.State.Bool() {
		for _, granted := range userConfig.MemberOf {
			err := pfh.pg.GrantRole(userName, granted)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pfh PgFgaHandler) handlePasswordUser(
	userConfig config.FgaUserConfig,
	userName string,
	options *pg.RoleOptionMap,
) (err error) {
	options.AddAbsolute(pg.RoleLogin)
	user, err := pg.NewRole(pfh.pg, userName, *options, userConfig.State)
	if err != nil {
		return err
	}
	// Note: if no password is set, it will be reset...
	err = user.SetPassword(userConfig.Password)
	if err != nil {
		return err
	}
	err = user.SetExpiry(userConfig.Expiry)
	if err != nil {
		return err
	}
	return nil
}

func (pfh PgFgaHandler) handleUsers() (err error) {
	for userName, userConfig := range pfh.config.UserConfig {
		options := pg.RoleOptionMap{}
		for _, optionName := range userConfig.Options {
			option := pg.RoleOption(optionName)
			if err = option.Validate(); err != nil {
				return err
			}
			options.AddAbsolute(option)
		}
		switch userConfig.Auth {
		case "ldap-group":
			if err = pfh.handleLdapGroup(userConfig, userName, options); err != nil {
				return err
			}
		case "ldap-user", "clientcert":
			if err = pfh.handleLdapUser(userConfig, userName, &options); err != nil {
				return err
			}
		case "password", "md5":
			if err = pfh.handlePasswordUser(userConfig, userName, &options); err != nil {
				return err
			}
		default:
			log.Fatalf("Invalid auth %s for user %s", userConfig.Auth, userName)
		}
	}
	return nil
}

func (pfh PgFgaHandler) handleDatabases() (err error) {
	return pfh.pg.CreateOrDropDatabases()
}

func (pfh PgFgaHandler) handleRoles() (err error) {
	for roleName, roleConfig := range pfh.config.Roles {
		options := pg.RoleOptionMap{}
		for _, optionName := range roleConfig.Options {
			option := pg.RoleOption(optionName)
			if err = option.Validate(); err != nil {
				return err
			}
			options[option] = option.Enabled()
		}
		role, err := pg.NewRole(pfh.pg, roleName, options, roleConfig.State)
		if err != nil {
			return err
		}
		for _, groupName := range roleConfig.MemberOf {
			group, err := pfh.pg.GetRole(groupName)
			if err != nil {
				return err
			}
			err = role.GrantRole(group)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pfh PgFgaHandler) handleSlots() (err error) {
	return pfh.pg.CreateOrDropSlots()
}
