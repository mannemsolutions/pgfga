// Package pg has all the logic to verify PostgreSQL state and apply changes
package pg

import (
	"context"
	"fmt"
	"os"
	"os/user"

	"github.com/jackc/pgx/v4"
)

// Conn is a smart PostgreSQL connection, which means that it has layers of methods
type Conn struct {
	connParams DSN
	conn       *pgx.Conn
}

// NewConn returns a connection with connection parameters set
func NewConn(connParams DSN) (c *Conn) {
	return &Conn{
		connParams: connParams,
	}
}

// DBName retrieves and returns the name of the database that Conn is connected to
func (c *Conn) DBName() (dbName string) {
	value, ok := c.connParams["dbname"]
	if ok {
		return value
	}
	value = os.Getenv("PGDATABASE")
	if value != "" {
		return value
	}
	return c.UserName()
}

// UserName retrieves and returns the name of the user that Conn is using for its connection to the database
func (c *Conn) UserName() (userName string) {
	value, ok := c.connParams["user"]
	if ok {
		return value
	}
	value = os.Getenv("PGUSER")
	if value != "" {
		return value
	}
	currentUser, err := user.Current()
	if err != nil {
		panic("cannot determine current user")
	}
	return currentUser.Username
}

// DSN returns a copy of the DSN
func (c *Conn) DSN() (dsn DSN) {
	return c.connParams.Clone()
}

// Connect can be used to connect to Postgres.
// If there already is an open connection, this just returns the connection.
// If not, it will instantiate a new pgx.Conn, connect to Postgres, and store it internally before returning it.
func (c *Conn) Connect() (err error) {
	if c.conn != nil {
		if !c.conn.IsClosed() {
			return nil
		}
		c.conn = nil
	}
	c.conn, err = pgx.Connect(context.Background(), c.DSN().String())
	if err != nil {
		c.conn = nil
		return err
	}
	return nil
}

func (c *Conn) runQueryExists(query string, args ...any) (exists bool, err error) {
	err = c.Connect()
	if err != nil {
		return false, err
	}
	var answer string
	err = c.conn.QueryRow(context.Background(), query, args...).Scan(&answer)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err == nil {
		return true, nil
	}
	return false, err
}

func (c *Conn) runQueryExec(query string, args ...any) (err error) {
	err = c.Connect()
	if err != nil {
		return err
	}
	_, err = c.conn.Exec(context.Background(), query, args...)
	return err
}

func (c *Conn) runQueryGetOneField(query string, args ...any) (answer string, err error) {
	err = c.Connect()
	if err != nil {
		return "", err
	}

	err = c.conn.QueryRow(context.Background(), query, args...).Scan(&answer)
	if err != nil {
		return "", fmt.Errorf("runQueryGetOneField (%s) failed: %v", query, err)
	}
	return answer, nil
}
