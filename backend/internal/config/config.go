// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration for the server.
type Config struct {
	// Addr is the TCP address the HTTP server listens on.
	Addr string `env:"CH_ADDR" envDefault:":8080"`

	// BaseURL is the externally reachable base URL (used for OIDC redirects,
	// email links, etc.). Example: https://assets.example.com
	BaseURL string `env:"CH_BASE_URL" envDefault:"http://localhost:8080"`

	// DataDir is the writable directory for the SQLite database and uploads.
	DataDir string `env:"CH_DATA_DIR" envDefault:"./data"`

	// LogLevel is one of: debug, info, warn, error.
	LogLevel string `env:"CH_LOG_LEVEL" envDefault:"info"`

	// SessionSecret signs session cookies.
	SessionSecret string `env:"CH_SESSION_SECRET"`

	// SessionCookieName is the HTTP cookie name carrying the session ID.
	SessionCookieName string `env:"CH_SESSION_COOKIE_NAME" envDefault:"ch_session"`

	// SessionTTLHours controls how long a login session remains valid.
	SessionTTLHours int `env:"CH_SESSION_TTL_HOURS" envDefault:"720"`

	// OIDCEnabled controls whether OIDC login is enabled.
	OIDCEnabled bool `env:"CH_OIDC_ENABLED" envDefault:"false"`

	// OIDC settings. These env vars take priority over values stored in the
	// admin UI settings table.
	OIDCIssuerURL         string `env:"CH_OIDC_ISSUER_URL"`
	OIDCClientID          string `env:"CH_OIDC_CLIENT_ID"`
	OIDCClientSecret      string `env:"CH_OIDC_CLIENT_SECRET"`
	OIDCRedirectURL       string `env:"CH_OIDC_REDIRECT_URL"`
	OIDCAdminGroup        string `env:"CH_OIDC_ADMIN_GROUP"`
	OIDCProviderName      string `env:"CH_OIDC_PROVIDER_NAME" envDefault:"OIDC"`
	OIDCAllowRegistration bool   `env:"CH_OIDC_ALLOW_REGISTRATION" envDefault:"true"`

	// Bootstrap admin user (optional). If set, startup ensures this admin exists.
	BootstrapAdminUsername string `env:"CH_BOOTSTRAP_ADMIN_USERNAME"`
	BootstrapAdminEmail    string `env:"CH_BOOTSTRAP_ADMIN_EMAIL"`
	BootstrapAdminPassword string `env:"CH_BOOTSTRAP_ADMIN_PASSWORD"`

	// TrustedProxies is a list of CIDRs/IPs whose X-Forwarded-* headers are trusted.
	TrustedProxies []string `env:"CH_TRUSTED_PROXIES" envSeparator:","`
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	cfg.LogLevel = strings.ToLower(cfg.LogLevel)

	abs, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}
	cfg.DataDir = abs

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.SessionTTLHours <= 0 {
		return fmt.Errorf("CH_SESSION_TTL_HOURS must be > 0")
	}
	// OIDC fields may alternatively be configured via the admin UI settings
	// table, so we only validate them when all three are set via env vars.
	if c.OIDCEnabled && (c.OIDCIssuerURL != "" || c.OIDCClientID != "" || c.OIDCRedirectURL != "") {
		if c.OIDCIssuerURL == "" || c.OIDCClientID == "" || c.OIDCRedirectURL == "" {
			return fmt.Errorf("CH_OIDC_ISSUER_URL, CH_OIDC_CLIENT_ID and CH_OIDC_REDIRECT_URL must all be set together when configuring OIDC via environment variables")
		}
	}
	return nil
}

// DBPath returns the absolute path to the SQLite database file.
func (c *Config) DBPath() string { return filepath.Join(c.DataDir, "capital-hub.db") }

// UploadsDir returns the absolute path to the uploads directory.
func (c *Config) UploadsDir() string { return filepath.Join(c.DataDir, "uploads") }
