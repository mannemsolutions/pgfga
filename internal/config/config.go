// Package config is used to define a yaml representation of the PgFga config
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pgvillage-tools/pgfga/internal/version"
	"github.com/pgvillage-tools/pgfga/pkg/ldap"
	"github.com/pgvillage-tools/pgfga/pkg/pg"
	"go.uber.org/zap/zapcore"

	"gopkg.in/yaml.v2"
)

/*
 * This module reads the config file and returns a config object with all entries from the config yaml file.
 */

const (
	envConfName     = "PGFGACONFIG"
	defaultConfFile = "/etc/pgfga/config.yaml"
)

// FgaGeneralConfig is a definition of the config yaml file that can be used by PgFga
type FgaGeneralConfig struct {
	LogLevel zapcore.Level `yaml:"loglevel"`
	RunDelay time.Duration `yaml:"run_delay"`
	Debug    bool          `yaml:"debug"`
}

// FgaUserConfig holds all generic config regarding PostgreSQL users to be managed with PgFga
type FgaUserConfig struct {
	Auth     string    `yaml:"auth"`
	BaseDN   string    `yaml:"ldapbasedn"`
	Filter   string    `yaml:"ldapfilter"`
	MemberOf []string  `yaml:"memberof"`
	Options  []string  `yaml:"options"`
	Expiry   time.Time `yaml:"expiry"`
	Password string    `yaml:"password"`
	State    pg.State  `yaml:"state"`
}

// FgaRoleConfig holds all config regarding PostgreSQL roles to be managed with PgFga
type FgaRoleConfig struct {
	Options  []string `yaml:"options"`
	MemberOf []string `yaml:"member"`
	State    pg.State `yaml:"state"`
}

// FgaConfig holds all config regarding PostgreSQL roles to be managed with PgFga
type FgaConfig struct {
	GeneralConfig FgaGeneralConfig         `yaml:"general"`
	StrictConfig  pg.StrictOptions         `yaml:"strict"`
	LdapConfig    ldap.Config              `yaml:"ldap"`
	PgDsn         pg.DSN                   `yaml:"postgresql_dsn"`
	DbsConfig     pg.Databases             `yaml:"databases"`
	UserConfig    map[string]FgaUserConfig `yaml:"users"`
	Roles         map[string]FgaRoleConfig `yaml:"roles"`
	Slots         []string                 `yaml:"replication_slots"`
}

// NewConfig will instantiate a new Config and return it
func NewConfig() (config FgaConfig, err error) {
	var configFile string
	var debug bool
	var displayVersion bool
	flag.BoolVar(&debug, "d", false, "Add debugging output")
	flag.BoolVar(&displayVersion, "v", false, "Show version information")
	flag.StringVar(&configFile, "c", os.Getenv(envConfName), "Path to configfile")

	flag.Parse()
	if displayVersion {
		fmt.Println(version.GetAppVersion())
		os.Exit(0)
	}
	if configFile == "" {
		configFile = defaultConfFile
	}
	configFile, err = filepath.EvalSymlinks(configFile)
	if err != nil {
		return config, err
	}

	// This only parsed as yaml, nothing else
	// #nosec
	yamlConfig, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlConfig, &config)
	config.GeneralConfig.Debug = config.GeneralConfig.Debug || debug
	return config, err
}
