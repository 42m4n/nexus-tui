package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"nexus-tui/internal/nexus"
)

func testClients(t *testing.T) map[string]*nexus.Client {
	t.Helper()
	cs := map[string]*nexus.Client{}
	for _, n := range []string{"a", "b"} {
		c, err := nexus.New("https://"+n+".example", false, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		cs[n] = c
	}
	return cs
}

func sendKey(m Model, key tea.KeyType) Model {
	next, _ := m.Update(tea.KeyMsg{Type: key})
	return next.(Model)
}

func TestSwitchProfile(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "old"}}
	m.status = "writable"

	m = sendKey(m, tea.KeyCtrlP)
	if m.screen != scrSwitch {
		t.Fatalf("screen after ctrl+p = %v, want scrSwitch", m.screen)
	}

	m = sendKey(m, tea.KeyDown)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s := next.(Model)
	if s.c != cs["b"] {
		t.Error("client after switch != profiles[b]")
	}
	if s.profile != "b" {
		t.Errorf("profile = %q, want %q", s.profile, "b")
	}
	if s.screen != scrBrowse {
		t.Errorf("screen = %v, want scrBrowse", s.screen)
	}
	if len(s.repos) != 0 {
		t.Errorf("repos = %v, want cleared", s.repos)
	}
	if s.status != "" {
		t.Errorf("status = %q, want cleared", s.status)
	}
	if cmd == nil {
		t.Error("cmd = nil, want reload batch")
	}
}

func TestSwitchProfileNoopOnCurrent(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.screen = scrSwitch
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s := next.(Model)
	if s.c != cs["a"] {
		t.Error("client changed on no-op switch")
	}
	if cmd != nil {
		t.Errorf("cmd = %v, want nil", cmd)
	}
}
