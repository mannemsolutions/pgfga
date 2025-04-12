package pg

import (
	"fmt"
	"strings"
)

// identifier returns the object name ready to be used in a sql query as an object name (e.a. select * from %s)
func identifier(objectName string) (escaped string) {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(objectName, "\"", "\"\""))
}

// quotedSqlValue uses proper quoting for values in SQL queries
func quotedSQLValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(objectName, "'", "''"))
}

// connectStringValue uses proper quoting for connect string values
func connectStringValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(objectName, "'", "\\'"))
}
