package ldap

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
)

const (
	executableBits = 0o111
)

// Credential is a structure to configure a credential.
// Credentials can be paased as a string, or from a file, and can be base64 encoded.
type Credential struct {
	Value  string `yaml:"value"`
	File   string `yaml:"file"`
	Base64 bool   `yaml:"base64"`
}

func isExecutable(filename string) (isExecutable bool, err error) {
	fi, err := os.Lstat(filename)
	if err != nil {
		return false, err
	}
	mode := fi.Mode()
	return mode&executableBits == executableBits, nil
}

func fromExecutable(filename string) (value string, err error) {
	// The intent is to give an option to use a 3rd party tool to retrieve a password.
	// Or a script to hash / unhash anyway you like
	// As such running an arbitrary command set as a parameter is sot of the point.
	// #nosec
	out, err := exec.Command(filename).Output()
	if err != nil {
		return "", nil
	}
	return string(out), nil
}

func fromFile(filename string) (value string, err error) {
	isExec, err := isExecutable(filename)
	if err != nil {
		return "", err
	}
	if isExec {
		return fromExecutable(filename)
	}
	// The intent is to give an option to retrieve a password from a file.
	// As such opening a file which name is set by a variable is sort of the point.
	// #nosec
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data[:]), nil
}

// GetCred is a method to retrieve the Credential, and return it's unencrypted string value (or an error).
func (c *Credential) GetCred() (string, error) {
	var err error
	if c.Value == "" && c.File == "" {
		return "", errors.New("either value or file must be set in a credential")
	}
	if c.Value == "" {
		if c.Value, err = fromFile(c.File); err != nil {
			return "", err
		}
	}
	if c.Value == "" {
		return "", errors.New("credential file is empty")
	}
	if c.Base64 {
		data, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			return "", err
		}
		c.Value = string(data)
		c.Base64 = false
		if c.Value == "" {
			return "", errors.New("empty credential after base64 decryption")
		}
	}
	return c.Value, nil
}
