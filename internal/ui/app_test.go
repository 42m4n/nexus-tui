package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

func TestTabCompletesCommand(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue("rep")
	m = sendKey(m, tea.KeyTab)
	if got := m.bar.Value(); got != "repos" {
		t.Errorf("bar = %q, want %q", got, "repos")
	}
}

func TestTabCompletionPreservesArg(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue("rep maven-extra")
	m = sendKey(m, tea.KeyTab)
	if got := m.bar.Value(); got != "repos maven-extra" {
		t.Errorf("bar = %q, want %q", got, "repos maven-extra")
	}
}

func TestTabNoMatchKeepsBar(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue(":zzz")
	m = sendKey(m, tea.KeyTab)
	if got := m.bar.Value(); got != ":zzz" {
		t.Errorf("bar = %q, want %q", got, ":zzz")
	}
}

func TestTabAmbiguousKeepsBar(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barCmd
	m.bar.SetValue("c")
	m = sendKey(m, tea.KeyTab)
	if got := m.bar.Value(); got != "c" {
		t.Errorf("bar = %q, want %q", got, "c")
	}
}

func TestTabFilterModeNoop(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.barMode = barFilter
	m.bar.SetValue("mav")
	m = sendKey(m, tea.KeyTab)
	if got := m.bar.Value(); got != "mav" {
		t.Errorf("bar = %q, want %q", got, "mav")
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

// plain strips ANSI escapes so header assertions see visible text.
func plain(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

func headerLine(m Model, w int) string {
	m.width = w
	return plain(m.header())
}

func TestHeaderRichData(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.status = "writable"
	m.nxVer = "3.86.2-01"
	m.checks = []checkRow{
		{name: "cpu", st: nexus.CheckResult{Healthy: true}},
		{name: "disk", st: nexus.CheckResult{Healthy: true}},
	}
	m.blobs = []nexus.BlobStore{{Name: "default", AvailableSpace: 5 << 30}}

	line := headerLine(m, 120)
	for _, want := range []string{"nx:3.86.2-01", "checks:2/2", "blobs:1/1", "5.0 GiB free", "writable", "writes:off"} {
		if !strings.Contains(line, want) {
			t.Errorf("header missing %q: %q", want, line)
		}
	}
}

func TestHeaderDegrades(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.status = "writable"
	// nxVer empty -> nx:-; no checks, no blobs, no ro
	line := headerLine(m, 120)
	if !strings.Contains(line, "nx:-") {
		t.Errorf("header missing fallback %q: %q", "nx:-", line)
	}
	if strings.Contains(line, "checks:") {
		t.Errorf("header shows checks without data: %q", line)
	}
	if strings.Contains(line, "blobs:") {
		t.Errorf("header shows blobs without data: %q", line)
	}
}

func TestHeaderFailureStates(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.status = "read-only"
	m.checks = []checkRow{
		{name: "cpu", st: nexus.CheckResult{Healthy: true}},
		{name: "disk", st: nexus.CheckResult{Healthy: false}},
	}
	m.blobs = []nexus.BlobStore{{Name: "default", AvailableSpace: 100 << 20}} // under 1 GiB

	line := headerLine(m, 120)
	for _, want := range []string{"checks:1/2", "blobs:0/1", "read-only"} {
		if !strings.Contains(line, want) {
			t.Errorf("header missing %q: %q", want, line)
		}
	}
}

func TestHeaderReadOnlyFrozen(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.status = "writable"
	m.roKnown = true
	m.readOnly = nexus.ReadOnlyState{Frozen: true, SummaryReason: "db locked"}

	line := headerLine(m, 120)
	if !strings.Contains(line, "RO:frozen(db locked)") {
		t.Errorf("header missing RO:frozen: %q", line)
	}
}

func TestHeaderNarrowDropsStorage(t *testing.T) {
	cs := testClients(t)
	m := New(cs, "a")
	m.status = "writable"
	m.nxVer = "3.86.2-01"
	m.checks = []checkRow{{name: "cpu", st: nexus.CheckResult{Healthy: true}}}
	m.blobs = []nexus.BlobStore{{Name: "default", AvailableSpace: 5 << 30}}

	wide := headerLine(m, 120)
	if !strings.Contains(wide, "blobs:1/1") {
		t.Fatalf("wide header missing blobs: %q", wide)
	}
	narrow := headerLine(m, 80)
	if strings.Contains(narrow, "blobs:") {
		t.Errorf("narrow header still shows blobs: %q", narrow)
	}
	if !strings.Contains(narrow, "checks:") {
		t.Errorf("narrow header dropped checks: %q", narrow)
	}
	if lipgloss.Width(m.header()) > 80 { // headerLine already set width; recompute plain width
		t.Errorf("narrow header exceeds 80 cols")
	}
}
