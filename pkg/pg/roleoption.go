package pg

import (
	"fmt"
	"strings"
)

// RoleOption is a string type representing a option that grants extra rights to a user
type RoleOption string

const (
	// RoleSuperUser determines whether the new role is a “superuser”, who can override all access restrictions within
	// the database. Superuser status is dangerous and should be used only when really needed. PgFga must be a
	// superuser to create a superusers. If not specified, NOSUPERUSER is the default.
	RoleSuperUser RoleOption = "SUPERUSER"
	// RoleLogin determines whether a role is allowed to log in; that is, whether the role can be given as the
	// initial session authorization name during client connection. A role having the LOGIN attribute can be thought of
	// as a user. Roles without this attribute are useful for managing database privileges, but are not users in the
	// usual sense of the word. If not specified, NOLOGIN is the default, except when CREATE ROLE is invoked through its
	// alternative spelling CREATE USER.
	RoleLogin RoleOption = "LOGIN"
	// RoleCreateRole determines whether a role will be permitted to create, alter, drop, comment on, and change the
	// security label for other roles. See role creation for more details about what capabilities are conferred by this
	// privilege. If not specified, NOCREATEROLE is the default.
	RoleCreateRole RoleOption = "CREATEROLE"
	// RoleCreateUser grants CREATEUSER, which si an alias to CREATEROLE
	RoleCreateUser RoleOption = "CREATEUSER"
	// RoleCreateDB defines a role's ability to create databases. If CREATEDB is specified, the role being defined will
	// be allowed to create new databases. Specifying NOCREATEDB will deny a role the ability to create databases.
	// If not specified, NOCREATEDB is the default. Only superuser roles or roles with CREATEDB can specify CREATEDB.
	RoleCreateDB RoleOption = "CREATEDB"
	// RoleInherit (17+) affects the membership inheritance status when this role is added as a member of another role,
	// both in this and future commands. Specifically, it controls the inheritance status of memberships added with this
	// command using the IN ROLE clause, and in later commands using the ROLE clause. It is also used as the default
	// inheritance status when adding this role as a member using the GRANT command. If not specified, INHERIT is the
	// default.
	//
	// In PostgreSQL versions before 16, inheritance was a role-level attribute that controlled all runtime membership
	// checks for that role.
	RoleInherit RoleOption = "INHERIT"
	// RoleReplication These clauses determine whether a role is a replication role. A role must have this attribute
	// (or be a superuser) in order to be able to connect to the server in replication mode (physical or logical
	// replication) and in order to be able to create or drop replication slots. A role having the REPLICATION attribute
	// is a very highly privileged role, and should only be used on roles actually used for replication. If not
	// specified, NOREPLICATION is the default. Only superuser roles or roles with REPLICATION can specify REPLICATION.
	RoleReplication RoleOption = "REPLICATION"
	// RoleBypassRLS These clauses determine whether a role bypasses every row-level security (RLS) policy.
	// NOBYPASSRLS is the default. Only superuser roles or roles with BYPASSRLS can specify BYPASSRLS.
	//
	// Note that pg_dump will set row_security to OFF by default, to ensure all contents of a table are dumped out.
	// If the user running pg_dump does not have appropriate permissions, an error will be returned. However, superusers
	// and the owner of the table being dumped always bypass RLS.
	RoleBypassRLS RoleOption = "BYPASSRLS"
)

var (
	// AllNormalRoleOptions is a list that contains all normal defined options
	AllNormalRoleOptions = RoleOptionList{
		RoleSuperUser,
		RoleLogin,
		RoleCreateRole,
		RoleCreateUser,
		RoleCreateDB,
		RoleInherit,
		RoleReplication,
		RoleBypassRLS,
	}
	// AllInvertedRoleOptions is a list that contains all inverted defined options
	AllInvertedRoleOptions = AllNormalRoleOptions.Inverted()
	// AllRoleOptions is a list that contains all normal and inverted RoleOptions
	AllRoleOptions = append(AllNormalRoleOptions, AllInvertedRoleOptions...)
	// AllValidRoleOptions is a map version of all normal and inverted RoleOptions
	AllValidRoleOptions = AllRoleOptions.ToValidMap()
)

// Validate will check if this is a valid option and return an error if it is, or nil if it isn't
func (opt RoleOption) Validate() error {
	_, exists := AllValidRoleOptions[opt]
	if !exists {
		return fmt.Errorf("%s is not a valid option", opt)
	}
	return nil
}

func (opt RoleOption) String() string {
	return string(opt)
}

// Absolute will return the normalized version of the RoleOption (e.a. strip NO if it starts with NO)
func (opt RoleOption) Absolute() RoleOption {
	inverted, _ := strings.CutPrefix(string(opt), "NO")
	return RoleOption(inverted)
}

// Enabled returns false if it starts with NO en true if it doesn't
func (opt RoleOption) Enabled() bool {
	return !strings.HasPrefix(string(opt), "NO")
}

// SQL returns the SQL part to add if you want to set this option for the specified role
func (opt RoleOption) SQL() (sql string) {
	if opt.Enabled() {
		if opt == RoleSuperUser {
			return "rolsuper"
		}
		return "rol" + strings.ToLower(string(opt))
	}
	return fmt.Sprintf("not %s", opt.Invert().SQL())
}

// Invert will return the inverted version of the RoleOption (e.a. strip NO if it starts with NO, else prefix with NO)
func (opt RoleOption) Invert() RoleOption {
	inverted, found := strings.CutPrefix(string(opt), "NO")
	if found {
		return RoleOption(inverted)
	}
	return "NO" + opt
}

// MarshalYAML marshals the enum as a quoted json string
func (opt RoleOption) MarshalYAML() (any, error) {
	return opt.String(), nil
}

// RoleOptionFromYAML converts a yaml string to the enum value
func RoleOptionFromYAML(unmarshal func(any) error) (*RoleOption, error) {
	var name string
	if err := unmarshal(&name); err != nil {
		return nil, err
	}
	tmpOpt := RoleOption(name)
	if err := tmpOpt.Validate(); err != nil {
		return nil, err
	}
	return &tmpOpt, nil
}

// func (ros RoleOptions)Join(sep string) (joined string) {
//     var strOptions []string
// 	   for _, option := range ros {
// 	       strOptions = append(strOptions, string(option))
// 	   }
// 	   return strings.Join(strOptions, sep)
// }
