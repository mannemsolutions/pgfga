package internal

import (
	"encoding/json"
	"fmt"
)

// Use this function to print a human readable version of the returned struct.
func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		_, err := fmt.Println(string(b))
		if err != nil {
			return err
		}
	}
	return
}
