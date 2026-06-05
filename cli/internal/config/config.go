package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigDir  = ".config/pc"
	ConfigName = "config.yaml"
	CacheName  = "cache.json"
)

type Config struct {
	Woodpecker    Woodpecker `yaml:"woodpecker"`
	PersonalCloud Personal   `yaml:"personal_cloud"`
	GitHub        GitHub     `yaml:"github"`
	Defaults      Defaults   `yaml:"defaults"`
	GHCR          GHCR       `yaml:"ghcr"`
	VM            VM         `yaml:"vm"`
}

type Woodpecker struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type Personal struct {
	Owner     string `yaml:"owner"`
	Repo      string `yaml:"repo"`
	LocalPath string `yaml:"local_path"`
}

type VM struct {
	SSH string `yaml:"ssh"`
}

type GitHub struct {
	Owner string `yaml:"owner"`
}

type Defaults struct {
	TailnetBase string `yaml:"tailnet_base"`
	Registry    string `yaml:"registry"`
}

type GHCR struct {
	Token string `yaml:"token"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDir, ConfigName), nil
}

func CachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ConfigDir, CacheName), nil
}

func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w (run from a configured machine or create config — see docs/pc-cli.md)", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.applyDefaults()
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.PersonalCloud.Owner == "" {
		c.PersonalCloud.Owner = c.GitHub.Owner
	}
	if c.PersonalCloud.Repo == "" {
		c.PersonalCloud.Repo = "personal-cloud"
	}
	if c.GitHub.Owner == "" {
		c.GitHub.Owner = c.PersonalCloud.Owner
	}
	if c.Defaults.Registry == "" {
		c.Defaults.Registry = "ghcr.io"
	}
	if c.Defaults.TailnetBase == "" {
		c.Defaults.TailnetBase = "example.ts.net"
	}
}

func (c *Config) Validate() error {
	if c.Woodpecker.URL == "" {
		return fmt.Errorf("woodpecker.url is required")
	}
	if c.Woodpecker.Token == "" {
		return fmt.Errorf("woodpecker.token is required")
	}
	if c.PersonalCloud.Owner == "" {
		return fmt.Errorf("personal_cloud.owner or github.owner is required")
	}
	return nil
}

func Example() string {
	return `woodpecker:
  url: http://100.x.x.x:8000
  token: YOUR_WOODPECKER_PERSONAL_TOKEN
personal_cloud:
  owner: your-github-username
  repo: personal-cloud
  local_path: ~/personal-cloud
github:
  owner: your-github-username
defaults:
  tailnet_base: example.ts.net
  registry: ghcr.io
ghcr:
  token: YOUR_GITHUB_PAT_WITH_PACKAGES
vm:
  ssh: deploy
`
}
