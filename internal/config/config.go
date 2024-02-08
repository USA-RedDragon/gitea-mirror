package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// GitHubAuthConfig is the configuration for the GitHub instance
type GitHubAuthConfig struct {
	EnterpriseURL  string `json:"enterprise-url"`
	AppID          uint   `json:"app-id"`
	InstallationID uint   `json:"app-install-id"`
	PrivateKeyPath string `json:"app-private-key-path"`
	Token          string `json:"token"`
	MirroringToken string `json:"mirroring-token"`
}

// GiteaAuthConfig is the configuration for the Gitea instance
type GiteaAuthConfig struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// Entity is the type of entity to mirror
type Entity string

var (
	User         Entity = "user"
	Organization Entity = "organization"
)

// FilterConfig is the configuration for filtering repositories
type FilterConfig struct {
	// Include is a list of regular expressions to include
	Include []string `json:"include"`
	// Exclude is a list of regular expressions to exclude
	Exclude []string `json:"exclude"`

	// OnlyArchived is a flag to only include archived repositories
	OnlyArchived bool `json:"only-archived"`
}

// MatchInclusion returns true if the name matches any of the inclusion patterns
func (f FilterConfig) MatchInclusion(name string) bool {
	if len(f.Include) == 0 {
		return true
	}
	for _, pattern := range f.Include {
		if match, _ := regexp.MatchString(pattern, name); match {
			return true
		}
	}
	return false
}

// MatchExclusion returns true if the name matches any of the exclusion patterns
func (f FilterConfig) MatchExclusion(name string) bool {
	if len(f.Exclude) == 0 {
		return false
	}
	for _, pattern := range f.Exclude {
		if match, _ := regexp.MatchString(pattern, name); match {
			return true
		}
	}
	return false
}

// MirrorFromEntityConfig is the configuration for a single GitHub entity to mirror
type MirrorFromEntityConfig struct {
	Type Entity `json:"type"`
	Name string `json:"name"`

	Filter FilterConfig `json:"filter"`
}

// MirrorToEntityConfig is the configuration for a single Gitea entity to mirror to
type MirrorToEntityConfig struct {
	Name string `json:"name"`
}

type MirrorConfig struct {
	// Prefix is an optional prefix to add to the repository name
	Prefix string
	// Suffix is an optional suffix to add to the repository name
	Suffix string

	// From is the source entity to mirror
	From MirrorFromEntityConfig
	// To is the destination entity to mirror to
	To MirrorToEntityConfig
}

// Config is the main configuration for the application
type Config struct {
	GitHubAuth GitHubAuthConfig `json:"github"`
	GiteaAuth  GiteaAuthConfig  `json:"gitea"`
	Mirrors    []MirrorConfig   `json:"mirrors"`
}

//nolint:golint,gochecknoglobals
var (
	ConfigFileKey           = "config"
	GitHubEnterpriseURLKey  = "github-enterprise-url"
	GitHubAppIDKey          = "github-app-id"
	GitHubInstallationIDKey = "github-install-id"
	GitHubPrivateKeyPathKey = "github-private-key-path"
	GitHubToken             = "github-token"
	GiteaURL                = "gitea-url"
	GiteaToken              = "gitea-token"
)

func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(ConfigFileKey, "c", "", "Config file path")
	cmd.Flags().String(GitHubEnterpriseURLKey, "", "GitHub Enterprise URL")
	cmd.Flags().Uint(GitHubAppIDKey, 0, "GitHub App ID")
	cmd.Flags().Uint(GitHubInstallationIDKey, 0, "GitHub App installation ID")
	cmd.Flags().String(GitHubPrivateKeyPathKey, "", "Path to the GitHub App private key")
	cmd.Flags().String(GitHubToken, "", "GitHub Token")
	cmd.Flags().String(GiteaURL, "", "Gitea URL")
	cmd.Flags().String(GiteaToken, "", "Gitea Token")
}

func (c *Config) Validate() error {
	// Config must have auth for gitea and github
	if c.GitHubAuth.Token == "" && c.GitHubAuth.AppID == 0 {
		return fmt.Errorf("GitHub Token or App ID is required")
	}

	// Gitea Token is required
	if c.GiteaAuth.Token == "" {
		return fmt.Errorf("Gitea Token is required")
	}

	// We're using GitHub PAT auth if Token is set
	isPATAuth := c.GitHubAuth.Token != ""

	// We're using GitHub App auth if App ID is set
	isAppAuth := c.GitHubAuth.AppID != 0

	// PAT and App auth are mutually exclusive
	if isPATAuth && isAppAuth {
		return fmt.Errorf("GitHub PAT and App auth are mutually exclusive")
	}

	// GitHub App ID is required if using GitHub App auth
	if isAppAuth && c.GitHubAuth.AppID == 0 {
		return fmt.Errorf("GitHub App ID is required")
	}

	// GitHub Installation ID is required if using GitHub App auth
	if isAppAuth && c.GitHubAuth.InstallationID == 0 {
		return fmt.Errorf("GitHub App installation ID is required")
	}

	// GitHub Private Key Path is required if using GitHub App auth
	if isAppAuth && c.GitHubAuth.PrivateKeyPath == "" {
		return fmt.Errorf("GitHub App private key path is required")
	}

	// GitHub Private Key Path must be a real file
	if isAppAuth {
		_, err := os.Stat(c.GitHubAuth.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("GitHub App private key path is invalid: %w", err)
		}
	}

	// GitHub Token is required if using GitHub PAT auth
	if isPATAuth && c.GitHubAuth.Token == "" {
		return fmt.Errorf("GitHub Token is required")
	}

	// If set, GitHub Enterprise URL must be a valid URL
	if c.GitHubAuth.EnterpriseURL != "" {
		_, err := url.Parse(c.GitHubAuth.EnterpriseURL)
		if err != nil {
			return fmt.Errorf("GitHub Enterprise URL is invalid: %w", err)
		}
	}

	// Mirroring Token is only required if not using GitHub App auth
	if isPATAuth && c.GitHubAuth.MirroringToken == "" {
		return fmt.Errorf("GitHub mirroring token is required")
	}

	// Gitea URL is required
	if c.GiteaAuth.URL == "" {
		return fmt.Errorf("Gitea URL is required")
	}

	// Gitea URL must be a valid URL
	_, err := url.Parse(c.GiteaAuth.URL)
	if err != nil {
		return fmt.Errorf("Gitea URL is invalid: %w", err)
	}

	// There must be at least one mirror
	if len(c.Mirrors) == 0 {
		return fmt.Errorf("at least one mirror is required")
	}

	// Each mirror must have at least one source and one destination
	for i, mirror := range c.Mirrors {
		if len(mirror.From.Name) == 0 {
			return fmt.Errorf("mirror %d has no source", i)
		}
		if strings.ToLower(string(mirror.From.Type)) != "user" && strings.ToLower(string(mirror.From.Type)) != "organization" {
			return fmt.Errorf("mirror %d has an invalid source type", i)
		}
		if len(mirror.To.Name) == 0 {
			return fmt.Errorf("mirror %d has no destination", i)
		}
	}

	return nil
}

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	var config Config

	// Load flags from envs
	ctx, cancel := context.WithCancelCause(cmd.Context())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if ctx.Err() != nil {
			return
		}
		optName := strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_")
		if val, ok := os.LookupEnv(optName); !f.Changed && ok {
			if err := f.Value.Set(val); err != nil {
				cancel(err)
			}
			f.Changed = true
		}
	})
	if ctx.Err() != nil {
		return &config, fmt.Errorf("failed to load env: %w", context.Cause(ctx))
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return &config, fmt.Errorf("failed to get config path: %w", err)
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return &config, fmt.Errorf("failed to read config: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return &config, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	if cmd.Flags().Changed(GitHubEnterpriseURLKey) {
		config.GitHubAuth.EnterpriseURL, err = cmd.Flags().GetString(GitHubEnterpriseURLKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get GitHub Enterprise URL: %w", err)
		}
	}

	if cmd.Flags().Changed(GitHubAppIDKey) {
		config.GitHubAuth.AppID, err = cmd.Flags().GetUint(GitHubAppIDKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get GitHub App ID: %w", err)
		}
	}

	if cmd.Flags().Changed(GitHubInstallationIDKey) {
		config.GitHubAuth.InstallationID, err = cmd.Flags().GetUint(GitHubInstallationIDKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get GitHub App installation ID: %w", err)
		}
	}

	if cmd.Flags().Changed(GitHubPrivateKeyPathKey) {
		config.GitHubAuth.PrivateKeyPath, err = cmd.Flags().GetString(GitHubPrivateKeyPathKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get GitHub App private key path: %w", err)
		}
	}

	if cmd.Flags().Changed(GitHubToken) {
		config.GitHubAuth.Token, err = cmd.Flags().GetString(GitHubToken)
		if err != nil {
			return &config, fmt.Errorf("failed to get GitHub Token: %w", err)
		}
	}

	if cmd.Flags().Changed(GiteaURL) {
		config.GiteaAuth.URL, err = cmd.Flags().GetString(GiteaURL)
		if err != nil {
			return &config, fmt.Errorf("failed to get Gitea URL: %w", err)
		}
	}

	if cmd.Flags().Changed(GiteaToken) {
		config.GiteaAuth.Token, err = cmd.Flags().GetString(GiteaToken)
		if err != nil {
			return &config, fmt.Errorf("failed to get Gitea Token: %w", err)
		}
	}

	err = config.Validate()
	if err != nil {
		return &config, fmt.Errorf("failed to validate config: %w", err)
	}

	return &config, nil
}
