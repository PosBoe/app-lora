// Package appconf holds the application configuration and reads it from a
// YAML file.
package appconf

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

// Config is the root of the YAML configuration file.
type Config struct {
	// Path is the file the config came from. It is not read from YAML.
	Path string `yaml:"-"`

	Site    SiteConfig    `yaml:"site"`
	Listen  ListenConfig  `yaml:"listen"`
	PG      PGConfig      `yaml:"pg"`
	Logging LoggingConfig `yaml:"logging"`
	Auth    AuthConfig    `yaml:"auth"`
}

// SiteConfig is the public address of the application.
type SiteConfig struct {
	// URL is the address users open in a browser.
	URL string `yaml:"url"`
}

// ListenConfig holds the addresses the servers listen on.
type ListenConfig struct {
	// HTTP is the listen address of the HTTP server, e.g. ":8080".
	HTTP string `yaml:"http"`
}

// PGConfig holds the PostgreSQL connection settings.
type PGConfig struct {
	// DSN is a libpq connection string or URL.
	DSN string `yaml:"dsn"`
}

// LoggingConfig selects the level and the format of the process log.
type LoggingConfig struct {
	// Level is one of debug, info, warn, error.
	Level string `yaml:"level"`
	// Format is text or json.
	Format string `yaml:"format"`
}

// AuthConfig holds the login and session settings.
type AuthConfig struct {
	// SessionTTL is how long a session lives after its last use.
	SessionTTL time.Duration `yaml:"session_ttl"`
	// LockoutAttempts is the number of failed logins in a row that locks a
	// login.
	LockoutAttempts int `yaml:"lockout_attempts"`
	// LockoutDuration is how long a locked login stays locked. 0 takes the
	// default; the lockout cannot be turned off.
	LockoutDuration time.Duration `yaml:"lockout_duration"`
}

// Load reads the config file at path. It expands environment variables
// written as $VAR or ${VAR}, applies defaults, and validates the result.
// An unknown key is an error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the operator's own config file
	if err != nil {
		return nil, fmt.Errorf("appconf: %w", err)
	}

	conf, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("appconf: %s: %w", path, err)
	}
	conf.Path = path
	return conf, nil
}

// Parse decodes a config from YAML data. Load calls it after it reads the
// file.
func Parse(data []byte) (*Config, error) {
	data = []byte(os.ExpandEnv(string(data)))

	conf := new(Config)
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.Strict())
	if err := dec.Decode(conf); err != nil {
		return nil, err
	}

	conf.setDefaults()
	if err := conf.validate(); err != nil {
		return nil, err
	}
	return conf, nil
}

func (c *Config) setDefaults() {
	if c.Listen.HTTP == "" {
		c.Listen.HTTP = ":8080"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
	if c.Auth.SessionTTL == 0 {
		c.Auth.SessionTTL = 30 * 24 * time.Hour
	}
	if c.Auth.LockoutAttempts == 0 {
		c.Auth.LockoutAttempts = 5
	}
	if c.Auth.LockoutDuration == 0 {
		c.Auth.LockoutDuration = 5 * time.Minute
	}
}

func (c *Config) validate() error {
	if c.PG.DSN == "" {
		return errors.New("pg.dsn is required")
	}
	if _, err := ParseLogLevel(c.Logging.Level); err != nil {
		return fmt.Errorf("logging.level: %w", err)
	}
	switch c.Logging.Format {
	case "text", "json":
	default:
		return fmt.Errorf("logging.format: got %q, want text or json", c.Logging.Format)
	}
	if c.Auth.SessionTTL < time.Hour {
		return fmt.Errorf("auth.session_ttl: got %s, want at least 1h", c.Auth.SessionTTL)
	}
	if c.Auth.LockoutAttempts < 1 {
		return fmt.Errorf("auth.lockout_attempts: got %d, want at least 1", c.Auth.LockoutAttempts)
	}
	if c.Auth.LockoutDuration < time.Second {
		return fmt.Errorf("auth.lockout_duration: got %s, want at least 1s", c.Auth.LockoutDuration)
	}
	return nil
}

// ParseLogLevel converts a level name to a slog.Level.
func ParseLogLevel(s string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("got %q, want debug, info, warn or error", s)
	}
	return level, nil
}
