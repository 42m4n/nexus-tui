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

// sendStr delivers a printable key ("d", ":", "/", "?", "o") or a named key
// ("enter", "esc", "up", "down").
func sendStr(m Model, s string) Model {
	var msg tea.KeyMsg
	switch s {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	next, _ := m.Update(msg)
	return next.(Model)
}

func topKind(m Model) viewKind { return m.stack[len(m.stack)-1].kind }

func TestDrillInReposToComps(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "repo-a"}, {Name: "repo-b"}}
	m = sendStr(m, "enter")
	if got := topKind(m); got != vComps {
		t.Fatalf("got %v, want vComps", got)
	}
	if got := m.stack[len(m.stack)-1].repo; got != "repo-a" {
		t.Errorf("got %q, want %q", got, "repo-a")
	}
}

func TestCtxSwitchFlow(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")

	m = sendKey(m, tea.KeyCtrlP)
	if got := topKind(m); got != vCtx {
		t.Fatalf("top after ctrl+p = %v, want vCtx", got)
	}
	m = sendStr(m, "down")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s := next.(Model)
	if s.profile != "b" {
		t.Errorf("profile = %q, want %q", s.profile, "b")
	}
	if s.c != cs["b"] {
		t.Error("client after switch != clients[b]")
	}
	if len(s.stack) != 1 || s.stack[0].kind != vRepos {
		t.Errorf("stack = %v, want [repos]", s.stack)
	}
	if len(s.repos) != 0 {
		t.Errorf("repos = %v, want cleared", s.repos)
	}
	if cmd == nil {
		t.Error("cmd = nil, want reload batch")
	}
}

func TestCommandBarOpensView(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue(":users")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s := next.(Model)
	if got := topKind(s); got != vUsers {
		t.Fatalf("got %v, want vUsers", got)
	}
	if cmd == nil {
		t.Error("cmd = nil, want users load")
	}
}

func TestCommandBarUnknown(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue(":nope")
	m = sendStr(m, "enter")
	if m.err == "" {
		t.Error("err = empty, want unknown command message")
	}
	if len(m.stack) != 1 {
		t.Errorf("stack len = %d, want 1 (no push)", len(m.stack))
	}
}

func TestFilterBarApplies(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "maven-central"}, {Name: "npm-hosted"}}
	m.barMode = barFilter
	m.bar.SetValue("maven")
	m = sendStr(m, "enter")
	if got := m.top().filter; got != "maven" {
		t.Errorf("got %q, want %q", got, "maven")
	}
	if n := len(m.displayOrder(m.top())); n != 1 {
		t.Errorf("got %d rows, want 1", n)
	}
}

func TestDescribeViaD(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "repo-a", Format: "maven", Type: "proxy"}}
	m = sendStr(m, "d")
	if got := topKind(m); got != vDescribe {
		t.Fatalf("got %v, want vDescribe", got)
	}
	if m.descTitle != "repo-a" {
		t.Errorf("got %q, want %q", m.descTitle, "repo-a")
	}
}

func TestDeleteNeedsCtrlD(t *testing.T) {
	cs := testClients(t)
	cs["a"].Writes = true
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "repo-a"}}
	m = sendStr(m, "d") // describe, not delete
	if m.confirm.active {
		t.Error("confirm active after d, want describe only")
	}
	m2 := New(cs, "a")
	m2.repos = []nexus.Repository{{Name: "repo-a"}}
	next, _ := m2.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	s := next.(Model)
	if !s.confirm.active {
		t.Error("confirm inactive after ctrl+d, want active")
	}
}

func TestDeleteGatedByWrites(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "repo-a"}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if s := next.(Model); s.err == "" {
		t.Error("err = empty, want writes-disabled message")
	}
}

func TestHelpToggle(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m = sendStr(m, "?")
	if !m.showHelp {
		t.Fatal("showHelp = false, want true")
	}
	m = sendStr(m, "esc")
	if m.showHelp {
		t.Error("showHelp = true, want false")
	}
}

func TestSortCycle(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m = sendStr(m, "o")
	if got := m.top().sortCol; got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
	n := len(columns(vRepos))
	for i := 0; i < n; i++ {
		m = sendStr(m, "o")
	}
	if got := m.top().sortCol; got != -1 {
		t.Errorf("got %d, want -1 (back to server order)", got)
	}
}

func TestEscPopsCrumbs(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "repo-a"}}
	m = sendStr(m, "enter")
	if len(m.stack) != 2 {
		t.Fatalf("stack len = %d, want 2", len(m.stack))
	}
	m = sendStr(m, "esc")
	if len(m.stack) != 1 {
		t.Errorf("stack len = %d, want 1", len(m.stack))
	}
}

func TestAlertSchedulesClear(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "repo-a"}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	s := next.(Model)
	if s.err == "" {
		t.Fatal("err = empty, want writes-disabled message")
	}
	if s.msgExpiry.IsZero() {
		t.Error("msgExpiry = zero, want set")
	}
}

func TestFilterBarLive(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.repos = []nexus.Repository{{Name: "maven-central"}, {Name: "npm-hosted"}}
	m = sendStr(m, "/")
	if m.barMode != barFilter {
		t.Fatal("barMode != barFilter after /")
	}
	m = sendStr(m, "m")
	if got := m.top().filter; got != "m" {
		t.Errorf("live filter got %q, want %q", got, "m")
	}
	m = sendStr(m, "a")
	m = sendStr(m, "v")
	if got := m.top().filter; got != "mav" {
		t.Errorf("live filter got %q, want %q", got, "mav")
	}
	if n := len(m.displayOrder(m.top())); n != 1 {
		t.Errorf("got %d rows after live filter, want 1", n)
	}
	// esc clears the live filter
	m = sendStr(m, "esc")
	if got := m.top().filter; got != "" {
		t.Errorf("filter after esc = %q, want empty", got)
	}
	if n := len(m.displayOrder(m.top())); n != 2 {
		t.Errorf("got %d rows after esc, want 2", n)
	}
}

func TestSearchDebounceFires(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue("s")
	m = sendStr(m, "enter") // :s opens search view, query focused
	if got := topKind(m); got != vSearch {
		t.Fatalf("top = %v, want vSearch", got)
	}
	m = sendStr(m, "maven") // typing bumps searchVer
	if m.searchVer == 0 {
		t.Fatal("searchVer = 0 after typing, want > 0")
	}
	tick := searchDebounceMsg{ver: m.searchVer, query: "maven"}
	next, cmd := m.Update(tick)
	if s := next.(Model); !s.loading {
		t.Error("loading = false after debounce msg, want true")
	} else if cmd == nil {
		t.Error("cmd = nil after debounce msg, want searchComps")
	}
	stale := searchDebounceMsg{ver: m.searchVer - 1, query: "mav"}
	if next, _ := m.Update(stale); next.(Model).loading {
		t.Error("stale debounce msg triggered loading")
	}
}
