package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Profile struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`
	Insecure bool   `yaml:"insecure"`
}

type Config struct {
	Current  string             `yaml:"current"`
	Profiles map[string]Profile `yaml:"profiles"`
}

func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(dir, "nexus-tui", "config.yaml")
}

// Load reads the config file and applies env overrides. A missing file is
// not an error; env vars may supply everything.
func Load(path string) (Config, error) {
	var c Config
	if path == "" {
		path = DefaultPath()
	}
	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}

	// Env overrides apply to the selected profile, or create "env".
	name := c.Current
	if name == "" {
		name = "env"
	}
	p := c.Profiles[name]
	if v := os.Getenv("NEXUS_URL"); v != "" {
		p.URL = v
	}
	if v := os.Getenv("NEXUS_USER"); v != "" {
		p.Username = v
	}
	if v := os.Getenv("NEXUS_PASS"); v != "" {
		p.Password = v
	}
	if v := os.Getenv("NEXUS_TOKEN"); v != "" {
		p.Token = v
	}
	if v := os.Getenv("NEXUS_INSECURE"); v == "1" || v == "true" {
		p.Insecure = true
	}
	c.Profiles[name] = p
	return c, nil
}

// Select returns the named profile, or the configured current one.
func (c Config) Select(name string) (Profile, error) {
	if name == "" {
		name = c.Current
	}
	if name == "" {
		name = "env"
	}
	p, ok := c.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	if p.URL == "" {
		return Profile{}, fmt.Errorf("profile %q has no url", name)
	}
	return p, nil
}
