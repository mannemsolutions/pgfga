package ldap_test

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/pgvillage-tools/pgfga/pkg/ldap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fileReadOnly = 0o0600
)

func TestCredential(t *testing.T) {
	const myFirstValue = "myval1"
	const mySecondValue = "myval2"
	myBase64EncryptedValue := base64.StdEncoding.EncodeToString([]byte(mySecondValue))
	tmpDir, err := os.MkdirTemp("", "Credential")
	if err != nil {
		panic(fmt.Errorf("unable to create temp dir: %w", err))
	}
	defer os.RemoveAll(tmpDir)
	myCredFile := path.Join(tmpDir, "my-creds-file")
	require.NoError(t, os.WriteFile(myCredFile, []byte(myFirstValue), fileReadOnly))
	myB64CredFile := path.Join(tmpDir, "my-b64-creds-file")
	require.NoError(t, os.WriteFile(myB64CredFile, []byte(myBase64EncryptedValue), fileReadOnly))
	for _, test := range []struct {
		value    string
		file     string
		base64   bool
		expected string
	}{
		{value: myFirstValue, expected: myFirstValue},
		{file: myCredFile, expected: myFirstValue},
		{value: myBase64EncryptedValue, base64: true, expected: mySecondValue},
		{file: myB64CredFile, base64: true, expected: mySecondValue},
	} {
		t.Logf("test values: %v", test)
		cred := ldap.Credential{
			Value:  test.value,
			File:   test.file,
			Base64: test.base64,
		}
		myCred, err := cred.GetCred()
		assert.NoError(t, err)
		assert.Equal(t, test.expected, myCred)
	}
}
