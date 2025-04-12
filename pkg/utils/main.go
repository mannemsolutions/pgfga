// Package utils holds some generic functions
package utils

import (
	"encoding/json"
	"fmt"
)

// PrettyPrint can print a human readable version of the returned struct.
func PrettyPrint(v any) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return nil
}
