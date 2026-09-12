package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("current: prod\nprofiles:\n  prod:\n    url: http://file\n    username: fileuser\n"), 0o600)

	t.Setenv("NEXUS_URL", "http://env")
	t.Setenv("NEXUS_PASS", "secret")
	os.Unsetenv("NEXUS_USER")

	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.Select("")
	if err != nil {
		t.Fatal(err)
	}
	if p.URL != "http://env" {
		t.Errorf("url = %q, want env override", p.URL)
	}
	if p.Username != "fileuser" {
		t.Errorf("username = %q, want file value when env unset", p.Username)
	}
	if p.Password != "secret" {
		t.Errorf("password = %q", p.Password)
	}
}

func TestMissingProfile(t *testing.T) {
	c := Config{}
	if _, err := c.Select("nope"); err == nil {
		t.Fatal("expected error for missing profile")
	}
}
