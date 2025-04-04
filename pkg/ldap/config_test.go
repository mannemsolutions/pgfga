package ldap_test

import (
	"fmt"
	"os"
	"testing"
)

func TestTimes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ShredTime")
	if err != nil {
		panic(fmt.Errorf("unable to create temp dir: %w", err))
	}
	defer os.RemoveAll(tmpDir)
}
