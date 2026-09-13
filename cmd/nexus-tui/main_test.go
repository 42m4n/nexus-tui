package main

import (
	"testing"

	"nexus-tui/internal/config"
)

func TestBuildClients(t *testing.T) {
	t.Run("skips profiles without url", func(t *testing.T) {
		cfg := config.Config{Profiles: map[string]config.Profile{
			"prod":    {URL: "https://a.example"},
			"staging": {URL: "https://b.example"},
			"nourl":   {Username: "x"},
		}}
		cs, err := buildClients(cfg, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(cs) != 2 {
			t.Errorf("len(clients) = %d, want 2", len(cs))
		}
		if _, ok := cs["nourl"]; ok {
			t.Error(`cs["nourl"] present, want skipped`)
		}
		if !cs["prod"].Writes {
			t.Error(`cs["prod"].Writes = false, want true`)
		}
	})

	t.Run("rejects invalid url", func(t *testing.T) {
		cfg := config.Config{Profiles: map[string]config.Profile{
			"bad": {URL: "://broken"},
		}}
		if _, err := buildClients(cfg, false); err == nil {
			t.Error("err = nil, want error for invalid url")
		}
	})
}
