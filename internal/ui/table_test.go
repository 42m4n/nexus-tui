package ui

import (
	"testing"
)

func TestResolveAlias(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		kind   viewKind
		arg    string
		filter string
		ok     bool
	}{
		{name: "repos", input: ":repos", kind: vRepos, ok: true},
		{name: "short rep", input: ":rep", kind: vRepos, ok: true},
		{name: "comp with repo", input: ":comp maven-central", kind: vComps, arg: "maven-central", ok: true},
		{name: "search with query", input: ":search log4j", kind: vSearch, arg: "log4j", ok: true},
		{name: "filter suffix", input: ":repos /maven", kind: vRepos, filter: "maven", ok: true},
		{name: "ctx", input: ":ctx", kind: vCtx, ok: true},
		{name: "health alias", input: ":checks", kind: vHealth, ok: true},
		{name: "privs alias", input: ":privs", kind: vPrivs, ok: true},
		{name: "unknown", input: ":nope", ok: false},
		{name: "empty", input: ":", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, arg, filter, ok := resolveAlias(tc.input)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if kind != tc.kind {
				t.Errorf("kind = %v, want %v", kind, tc.kind)
			}
			if arg != tc.arg {
				t.Errorf("arg = %q, want %q", arg, tc.arg)
			}
			if filter != tc.filter {
				t.Errorf("filter = %q, want %q", filter, tc.filter)
			}
		})
	}
}

func TestMatchFilter(t *testing.T) {
	cells := []string{"maven-central", "proxy"}
	cases := []struct {
		name   string
		filter string
		want   bool
	}{
		{name: "empty matches", filter: "", want: true},
		{name: "substring", filter: "maven", want: true},
		{name: "case insensitive", filter: "MAVEN", want: true},
		{name: "regex", filter: "mav.n-.*", want: true},
		{name: "no match", filter: "npm", want: false},
		{name: "bad regex falls back", filter: "maven(", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchFilter(cells, tc.filter); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilterSort(t *testing.T) {
	rows := [][]string{
		{"npm-hosted", "20"},
		{"maven-central", "100"},
		{"maven-snap", "3"},
	}
	order := filterSort(rows, "maven", 0, true)
	if len(order) != 2 {
		t.Fatalf("got %d rows, want 2", len(order))
	}
	if rows[order[0]][0] != "maven-central" || rows[order[1]][0] != "maven-snap" {
		t.Errorf("got %v, want sorted [maven-central maven-snap]", order)
	}
	num := filterSort(rows, "", 1, true)
	if rows[num[0]][1] != "3" || rows[num[1]][1] != "20" || rows[num[2]][1] != "100" {
		t.Errorf("numeric asc wrong: %v", num)
	}
	desc := filterSort(rows, "", 1, false)
	if rows[desc[0]][1] != "100" {
		t.Errorf("numeric desc first = %s, want 100", rows[desc[0]][1])
	}
}
