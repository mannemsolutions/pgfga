// Package ldap takes care of all communication with the ldap server
package ldap

// Config is a struct that can hold all ldap config
type Config struct {
	Usr        Credential `yaml:"user"`
	Pwd        Credential `yaml:"password"`
	Servers    []string   `yaml:"servers"`
	MaxRetries int        `yaml:"conn_retries"`
}

func (c *Config) setDefaults() {
	if c.MaxRetries < 1 {
		c.MaxRetries = 1
	}
}

func (c Config) user() (user string, err error) {
	user, err = c.Usr.GetCred()
	if err != nil {
		return "", err
	}
	return user, nil
}

func (c Config) password() (pwd string, err error) {
	pwd, err = c.Pwd.GetCred()
	if err != nil {
		return "", err
	}
	return pwd, nil
}
