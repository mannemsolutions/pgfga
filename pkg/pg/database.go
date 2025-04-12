package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

// Databases is a map of all known Database objects
type Databases map[string]*Database

// Database is a struct that can hold database information
type Database struct {
	// for DB's created from yaml, handler and name are set by the pg.Handler
	handler *Handler
	name    string
	// conn is created from handler when required
	conn       *Conn
	Owner      string     `yaml:"Owner"`
	Extensions extensions `yaml:"extensions"`
	State      State      `yaml:"state"`
}

// newDatabase can be used to create a new Database object
func newDatabase(handler *Handler, name string, owner string) (d *Database) {
	db, exists := handler.databases[name]
	if exists {
		if db.Owner != owner {
			log.Debugf("Warning: DB %s already exists with different Owner %s. Changing to Owner %s.", name,
				db.Owner, owner)
			db.Owner = owner
		}
		return db
	}
	d = &Database{
		handler:    handler,
		name:       name,
		Owner:      owner,
		Extensions: make(extensions),
	}
	d.setDefaults()
	handler.databases[name] = d
	return d
}

// setDefaults is called to set all defaults for databases created from yaml
func (d *Database) setDefaults() {
	if d.Owner == "" {
		d.Owner = d.name
	}
	for name, ext := range d.Extensions {
		ext.db = d
		ext.name = name
	}
}

func (d *Database) getDbConnection() (c *Conn) {
	if d.conn != nil {
		return d.conn
	}
	// not yet initialized. Let's initialize
	if d.handler.conn.DBName() == d.name {
		d.conn = d.handler.conn
		return d.conn
	}

	connParams := map[string]string{}
	for key, value := range d.handler.conn.connParams {
		connParams[key] = value
	}
	connParams["dbname"] = d.name
	d.conn = NewConn(connParams)
	return d.conn
}

func (d *Database) drop() (err error) {
	ph := d.handler
	if !ph.strictOptions.Databases {
		log.Infof("skipping drop of database %s (not running with strict option for databases", d.name)
		return nil
	}
	exists, err := ph.conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if exists {
		err = ph.conn.runQueryExec(fmt.Sprintf("drop database %s", identifier(d.name)))
		if err != nil {
			return err
		}
		log.Infof("Database '%s' successfully dropped", d.name)
	}
	d.State = Absent
	return nil
}

func (d Database) create() (err error) {
	ph := d.handler

	exists, err := ph.conn.runQueryExists("SELECT datname FROM pg_database WHERE datname = $1", d.name)
	if err != nil {
		return err
	}
	if !exists {
		err = ph.conn.runQueryExec(fmt.Sprintf("CREATE DATABASE %s", identifier(d.name)))
		if err != nil {
			return err
		}
		log.Infof("Database '%s' successfully created", d.name)
	}
	exists, err = ph.conn.runQueryExists(
		`SELECT datname
		FROM pg_database db
		INNER JOIN pg_roles rol
		ON db.datdba = rol.oid
		WHERE datname = $1
		AND rolname = $2`,
		d.name,
		d.Owner,
	)
	if err != nil {
		return err
	}
	if !exists {
		// First make sure role exists
		_, err = d.handler.GetRole(d.Owner)
		if err != nil {
			return err
		}
		// Then set Owner
		err = ph.conn.runQueryExec(
			fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", identifier(d.name), identifier(d.Owner)),
		)
		if err != nil {
			return err
		}
		log.Infof("Database Owner successfully altered to '%s' on '%s'",
			d.Owner, d.name)
	}
	err = d.createOrDropExtensions()
	if err != nil {
		return err
	}
	err = ph.GrantRole(d.Owner, "opex")
	if err != nil {
		return err
	}
	readOnlyRoleName := fmt.Sprintf("%s_readonly", d.name)
	err = ph.GrantRole(readOnlyRoleName, "readonly")
	if err != nil {
		return err
	}
	return d.setReadOnlyGrants(readOnlyRoleName)
}

func (d Database) setReadOnlyGrants(readOnlyRoleName string) (err error) {
	c := d.getDbConnection()
	err = c.Connect()
	if err != nil {
		return err
	}
	var schema string
	var schemas []string
	query := `select distinct schemaname from pg_tables
              where schemaname not in ('pg_catalog','information_schema')
			  and schemaname||'.'||tablename not in (SELECT table_schema||'.'||table_name
                  FROM information_schema.role_table_grants
                  WHERE grantee = $1 and privilege_type = 'SELECT')`
	row := c.conn.QueryRow(context.Background(), query, readOnlyRoleName)
	for {
		scanErr := row.Scan(&schema)
		if scanErr == pgx.ErrNoRows {
			break
		} else if scanErr != nil {
			return fmt.Errorf("error getting ReadOnly grants (qry: %s, err %s)", query, err)
		}
		schemas = append(schemas, schema)
	}
	for _, schema := range schemas {
		err = c.runQueryExec(fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA %s TO %s", identifier(schema),
			identifier(readOnlyRoleName)))
		if err != nil {
			return err
		}
		log.Infof("successfully granted SELECT ON ALL TABLES in schema '%s' in DB '%s' to '%s'",
			schema, d.name, readOnlyRoleName)
	}
	return nil
}

/*
func (d *Database) addExtension(name string, schema string, version string) (e *extension, err error) {
	e, err = newExtension(d, name, schema, version)
	if err != nil {
		return nil, err
	}
	d.Extensions[name] = e
	return e, nil
}
*/

func (d *Database) createOrDropExtensions() (err error) {
	for _, e := range d.Extensions {
		if e.State.Bool() {
			err = e.create()
		} else {
			err = e.drop()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
