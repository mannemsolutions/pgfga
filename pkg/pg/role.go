package pg

import (
	"context"
	"time"

	// md5 is weak, but it is still an accepted password algorithm in Postgres.
	// #nosec
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

const (
	md5PasswordLength = 35
	md5PasswordPrefix = "md5"
)

// Roles is a map of all roles that should be created
type Roles map[string]Role

// Role is a struct to hold all important info about one PostgreSQL role
type Role struct {
	handler *Handler
	name    string
	options RoleOptionMap
	State   State
}

// NewRole returns a new Role object
func NewRole(handler *Handler, name string, options RoleOptionMap, state State) (r *Role, err error) {
	myRole, exists := handler.roles[name]
	if exists {
		if myRole.State.Bool() != state.Bool() {
			if handler.strictOptions.Users {
				return nil, fmt.Errorf("cannot change state from %s to %s for existing Role %s", state.String(),
					myRole.State.String(), name)
			}
		}
		myRole.options = myRole.options.AbsoluteMerge(options)
		return &myRole, nil
	}
	r = &Role{
		handler: handler,
		name:    name,
		options: options,
		State:   state,
	}
	if state.Bool() {
		err = r.create()
	} else {
		err = r.drop()
	}
	if err != nil {
		return r, err
	}
	handler.roles[name] = *r
	return r, nil
}

func (r *Role) drop() (err error) {
	ph := r.handler
	c := ph.conn
	if !ph.strictOptions.Users {
		log.Infof("not dropping user/Role %s (config.strict.Roles is not True)", r.name)
		return nil
	}
	existsQuery := "SELECT rolname FROM pg_Roles WHERE rolname = $1 AND rolname != CURRENT_USER"
	exists, err := c.runQueryExists(existsQuery, r.name)
	if err != nil {
		return err
	}
	if !exists {
		delete(r.handler.roles, r.name)
		return nil
	}
	var dbname string
	var newOwner string
	query := `select db.datname, o.rolname as newOwner from pg_database db inner join 
			  pg_Roles o on db.datdba = o.oid where db.datname != 'template0'`
	row := c.conn.QueryRow(context.Background(), query)
	for {
		scanErr := row.Scan(&dbname, &newOwner)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		dbConn := ph.getDb(dbname).getDbConnection()
		err = dbConn.runQueryExec(fmt.Sprintf("REASSIGN OWNED BY %s TO %s", identifier(r.name), identifier(newOwner)))
		if err != nil {
			return err
		}
		log.Debugf("Reassigned ownership from '%s' to '%s' in db '%s'", r.name, newOwner, dbname)
	}
	err = c.runQueryExec(fmt.Sprintf("DROP ROLE %s", identifier(r.name)))
	if err != nil {
		return err
	}
	r.State = Absent
	log.Infof("Role '%s' successfully dropped", r.name)
	return nil
}

func (r Role) create() (err error) {
	c := r.handler.conn
	exists, err := c.runQueryExists("SELECT rolname FROM pg_Roles WHERE rolname = $1", r.name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("CREATE ROLE %s", identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully created", r.name)
	}
	for option := range r.options {
		err = r.setRoleOption(option)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r Role) setRoleOption(option RoleOption) (err error) {
	c := r.handler.conn
	exists, err := c.runQueryExists("SELECT rolname FROM pg_Roles WHERE rolname = $1 AND "+option.SQL(), r.name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER ROLE %s WITH "+option.String(), identifier(r.name)))
		if err != nil {
			return err
		}
		log.Debugf("Role '%s' successfully altered with option '%s'", r.name, option)
	}
	return nil
}

// GrantRole can be used to grant a Role to another Role.
func (r Role) GrantRole(grantedRole *Role) (err error) {
	c := r.handler.conn
	checkQry := `select granted.rolname granted_Role 
		from pg_auth_members auth inner join pg_Roles 
		granted on auth.Roleid = granted.oid inner join pg_Roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2`
	exists, err := c.runQueryExists(checkQry, grantedRole.name, r.name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.runQueryExec(fmt.Sprintf("GRANT %s TO %s", identifier(grantedRole.name), identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully granted to user '%s'", grantedRole.name, r.name)
	} else {
		log.Debugf("Role '%s' already granted to user '%s'", grantedRole.name, r.name)
	}
	return nil
}

// RevokeRole can be used to revoke a Role from another Role.
func (r Role) RevokeRole(roleName string) (err error) {
	c := r.handler.conn
	checkQry := `select granted.rolname granted_Role, grantee.rolname 
		grantee_Role from pg_auth_members auth inner join pg_Roles 
		granted on auth.Roleid = granted.oid inner join pg_Roles 
		grantee on auth.member = grantee.oid where 
		granted.rolname = $1 and grantee.rolname = $2 and grantee.rolname != CURRENT_USER`
	exists, err := c.runQueryExists(checkQry, roleName, r.name)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("REVOKE %s FROM %s", identifier(roleName), identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("Role '%s' successfully revoked from user '%s'", roleName, r.name)
	}
	return nil
}

// SetPassword can be used to set a password for a user.
func (r Role) SetPassword(password string) (err error) {
	if password == "" {
		return r.ResetPassword()
	}
	var hashedPassword string
	if len(password) == md5PasswordLength && strings.HasPrefix(password, md5PasswordPrefix) {
		hashedPassword = password
	} else {
		// #nosec
		hashedPassword = fmt.Sprintf("%s%x", md5PasswordPrefix, md5.Sum([]byte(password+r.name)))
	}
	c := r.handler.conn
	checkQry := `
	SELECT rolname 
	FROM pg_Roles 
	WHERE rolname = $1
		AND rolname NOT IN (
			SELECT usename 
			FROM pg_shadow 
			WHERE usename = $1
			AND COALESCE(passwd, '') = $2);`
	exists, err := c.runQueryExists(checkQry, r.name, hashedPassword)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER ROLE %s WITH ENCRYPTED PASSWORD %s", identifier(r.name),
			quotedSQLValue(hashedPassword)))
		if err != nil {
			return err
		}
		log.Infof("successfully set new password for user '%s'", r.name)
	}
	return nil
}

// ResetPassword can be used to reset the password of a PostgreSQL user
func (r Role) ResetPassword() (err error) {
	c := r.handler.conn
	checkQry := `SELECT usename FROM pg_shadow WHERE usename = $1
	AND Passwd IS NOT NULL AND usename != CURRENT_USER`
	exists, err := c.runQueryExists(checkQry, r.name)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER USER %s WITH PASSWORD NULL", identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("successfully removed password for user '%s'", r.name)
	}
	return nil
}

// SetExpiry can be used to define expiry on a PostgreSQL user
func (r Role) SetExpiry(expiry time.Time) (err error) {
	if expiry.IsZero() {
		return r.ResetExpiry()
	}
	formattedExpiry := expiry.Format(time.RFC3339)

	c := r.handler.conn
	checkQry := `SELECT rolname FROM pg_Roles where rolname = $1 AND (rolvaliduntil IS NULL OR rolvaliduntil != $2);`
	exists, err := c.runQueryExists(checkQry, r.name, formattedExpiry)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER ROLE %s VALID UNTIL %s", identifier(r.name),
			quotedSQLValue(formattedExpiry)))
		if err != nil {
			return err
		}
		log.Infof("successfully set new expiry for user '%s'", r.name)
	}
	return nil
}

// ResetExpiry can be used to reset the expiry of a PostgreSQL User
func (r Role) ResetExpiry() (err error) {
	c := r.handler.conn
	checkQry := `SELECT rolname
	FROM pg_Roles
	WHERE rolname = $1
	AND rolvaliduntil IS NOT NULL
	AND rolvaliduntil != 'infinity';`
	exists, err := c.runQueryExists(checkQry, r.name)
	if err != nil {
		return err
	}
	if exists {
		err = c.runQueryExec(fmt.Sprintf("ALTER ROLE %s VALID UNTIL 'infinity'", identifier(r.name)))
		if err != nil {
			return err
		}
		log.Infof("successfully reset expiry for user '%s'", r.name)
	}
	return nil
}
