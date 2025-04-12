package pg

import (
	"fmt"
	"strings"
)

// DSN can hold all connection parameters
type DSN map[string]string

// String joins all key/value pairs into a space separated connection string
func (dsn DSN) String() string {
	var pairs []string
	for key, value := range dsn {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, connectStringValue(value)))
	}
	return strings.Join(pairs[:], " ")
}

// Clone returns a copy of this DSN
func (dsn DSN) Clone() DSN {
	clone := DSN{}
	for key, value := range dsn {
		clone[key] = value
	}
	return clone
}
