package ui

import (
	"fmt"
	"strconv"
	"strings"
)

// viewKind is one k9s-style resource view.
type viewKind int

const (
	vRepos viewKind = iota
	vComps
	vSearch
	vTasks
	vUsers
	vRoles
	vPrivs
	vBlobs
	vHealth
	vCtx
	vDescribe
)

// viewState is one crumbs-stack entry. sel/filter/sort are per-view.
type viewState struct {
	kind    viewKind
	title   string
	repo    string // vComps: drilled-in repository
	sel     int
	filter  string
	sortCol int // -1 = unsorted
	sortAsc bool
}

func newView(kind viewKind, title string) viewState {
	return viewState{kind: kind, title: title, sortCol: -1, sortAsc: true}
}

func (k viewKind) defaultTitle() string {
	switch k {
	case vRepos:
		return "repos"
	case vComps:
		return "components"
	case vSearch:
		return "search"
	case vTasks:
		return "tasks"
	case vUsers:
		return "users"
	case vRoles:
		return "roles"
	case vPrivs:
		return "privileges"
	case vBlobs:
		return "blobstores"
	case vHealth:
		return "health"
	case vCtx:
		return "contexts"
	case vDescribe:
		return "describe"
	}
	return ""
}

// resolveAlias maps a :command to a view with an optional arg (repo, query)
// and an optional /filter suffix, e.g. ":repos /maven".
func resolveAlias(input string) (kind viewKind, arg, filter string, ok bool) {
	s := strings.TrimSpace(strings.TrimPrefix(input, ":"))
	filter = ""
	if i := strings.Index(s, "/"); i >= 0 {
		filter = strings.TrimSpace(s[i+1:])
		s = strings.TrimSpace(s[:i])
	}
	f := strings.Fields(s)
	if len(f) == 0 {
		return vRepos, "", "", false
	}
	if len(f) > 1 {
		arg = strings.Join(f[1:], " ")
	}
	switch strings.ToLower(f[0]) {
	case "repos", "rep", "repositories", "repo":
		return vRepos, "", filter, true
	case "comp", "comps", "components":
		return vComps, arg, filter, true
	case "search", "find", "s":
		return vSearch, arg, filter, true
	case "tasks", "task":
		return vTasks, "", filter, true
	case "users", "user":
		return vUsers, "", filter, true
	case "roles", "role":
		return vRoles, "", filter, true
	case "privs", "privileges", "priv":
		return vPrivs, "", filter, true
	case "blobs", "blob", "stores", "blobstores":
		return vBlobs, "", filter, true
	case "health", "checks", "status":
		return vHealth, "", filter, true
	case "ctx", "contexts", "profiles", "switch":
		return vCtx, "", filter, true
	}
	return vRepos, "", "", false
}

// columns returns the header labels for a view.
func columns(k viewKind) []string {
	switch k {
	case vRepos:
		return []string{"NAME", "FORMAT", "TYPE", "URL"}
	case vComps:
		return []string{"GROUP", "NAME", "VERSION", "ASSETS"}
	case vSearch:
		return []string{"GROUP", "NAME", "VERSION", "REPOSITORY"}
	case vTasks:
		return []string{"NAME", "STATE", "TYPE", "NEXT RUN"}
	case vUsers:
		return []string{"USERID", "EMAIL", "SOURCE", "STATUS"}
	case vRoles:
		return []string{"NAME", "DESCRIPTION"}
	case vPrivs:
		return []string{"NAME", "TYPE", "DESCRIPTION"}
	case vBlobs:
		return []string{"NAME", "TYPE", "BLOBS", "SIZE", "FREE"}
	case vHealth:
		return []string{"CHECK", "STATUS", "MESSAGE"}
	case vCtx:
		return []string{"PROFILE", "CURRENT"}
	}
	return nil
}

// rows builds one string cell per column, 1:1 with the underlying slice.
func (m Model) rows(k viewKind) [][]string {
	switch k {
	case vRepos:
		out := make([][]string, 0, len(m.repos))
		for _, r := range m.repos {
			out = append(out, []string{r.Name, r.Format, r.Type, r.URL})
		}
		return out
	case vComps:
		out := make([][]string, 0, len(m.comps))
		for _, c := range m.comps {
			out = append(out, []string{trim(c.Group, 30), trim(c.Name, 40), trim(c.Version, 20), strconv.Itoa(len(c.Assets))})
		}
		return out
	case vSearch:
		out := make([][]string, 0, len(m.comps))
		for _, c := range m.comps {
			out = append(out, []string{trim(c.Group, 30), trim(c.Name, 40), trim(c.Version, 20), trim(c.Repository, 25)})
		}
		return out
	case vTasks:
		out := make([][]string, 0, len(m.tasks))
		for _, t := range m.tasks {
			out = append(out, []string{trim(t.Name, 32), trim(t.CurrentState, 12), trim(t.Type, 25), trim(orDash(t.NextRun), 22)})
		}
		return out
	case vUsers:
		out := make([][]string, 0, len(m.users))
		for _, u := range m.users {
			out = append(out, []string{trim(u.UserID, 20), trim(u.Email, 30), trim(u.Source, 10), trim(u.Status, 10)})
		}
		return out
	case vRoles:
		out := make([][]string, 0, len(m.roles))
		for _, r := range m.roles {
			out = append(out, []string{trim(r.Name, 28), trim(r.Description, 70)})
		}
		return out
	case vPrivs:
		out := make([][]string, 0, len(m.privs))
		for _, p := range m.privs {
			out = append(out, []string{trim(p.Name, 32), trim(p.Type, 18), trim(p.Description, 50)})
		}
		return out
	case vBlobs:
		out := make([][]string, 0, len(m.blobs))
		for _, b := range m.blobs {
			out = append(out, []string{b.Name, b.Type,
				strconv.FormatInt(b.BlobCount, 10),
				strconv.FormatInt(b.TotalSize, 10),
				humanBytes(b.AvailableSpace)})
		}
		return out
	case vHealth:
		out := make([][]string, 0, len(m.checks))
		for _, c := range m.checks {
			st := "OK"
			if !c.st.Healthy {
				st = "FAIL"
			}
			out = append(out, []string{c.name, st, trim(squashHTML(c.st.Message), 70)})
		}
		return out
	case vCtx:
		out := make([][]string, 0, len(m.profiles))
		for _, p := range m.profiles {
			cur := ""
			if p == m.profile {
				cur = "*"
			}
			out = append(out, []string{p, cur})
		}
		return out
	}
	return nil
}

// displayOrder maps the current view through filter+sort.
func (m Model) displayOrder(vs *viewState) []int {
	return filterSort(m.rows(vs.kind), vs.filter, vs.sortCol, vs.sortAsc)
}

// selectedIndex maps the cursor to the underlying slice index, or -1.
func (m Model) selectedIndex(vs *viewState) int {
	order := m.displayOrder(vs)
	vs.sel = clampSel(vs.sel, len(order))
	if len(order) == 0 {
		return -1
	}
	return order[vs.sel]
}

// describe builds detail lines for the selected row of a view.
func (m Model) describe(vs *viewState) (string, []string) {
	i := m.selectedIndex(vs)
	if i < 0 {
		return "", []string{"(empty)"}
	}
	switch vs.kind {
	case vRepos:
		r := m.repos[i]
		return r.Name, []string{
			"name:    " + r.Name,
			"format:  " + r.Format,
			"type:    " + r.Type,
			"url:     " + r.URL,
			fmt.Sprintf("size:    %d", r.Size),
		}
	case vComps, vSearch:
		c := m.comps[i]
		lines := []string{
			"group:      " + c.Group,
			"name:       " + c.Name,
			"version:    " + c.Version,
			"repository: " + c.Repository,
			"format:     " + c.Format,
			"id:         " + c.ID,
			fmt.Sprintf("assets (%d):", len(c.Assets)),
		}
		for _, a := range c.Assets {
			lines = append(lines, "  - "+a.Path)
		}
		return c.Name + ":" + c.Version, lines
	case vTasks:
		t := m.tasks[i]
		return t.Name, []string{
			"name:    " + t.Name,
			"type:    " + t.Type,
			"state:   " + t.CurrentState,
			"last:    " + orDash(t.LastRun) + " / " + orDash(t.LastRunResult),
			"next:    " + orDash(t.NextRun),
			"message: " + trim(t.Message, 100),
		}
	case vUsers:
		u := m.users[i]
		return u.UserID, []string{
			"userid: " + u.UserID,
			"name:   " + u.FirstName + " " + u.LastName,
			"email:  " + u.Email,
			"source: " + u.Source,
			"status: " + u.Status,
		}
	case vRoles:
		r := m.roles[i]
		lines := []string{"name: " + r.Name, "description: " + r.Description, "privileges:"}
		lines = append(lines, r.Privileges...)
		return r.Name, lines
	case vPrivs:
		p := m.privs[i]
		return p.Name, []string{"name: " + p.Name, "type: " + p.Type, "description: " + p.Description}
	case vBlobs:
		b := m.blobs[i]
		return b.Name, []string{
			"name:  " + b.Name,
			"type:  " + b.Type,
			fmt.Sprintf("blobs: %d  size: %s  free: %s", b.BlobCount, humanBytes(b.TotalSize), humanBytes(b.AvailableSpace)),
		}
	case vHealth:
		c := m.checks[i]
		return c.name, []string{"check: " + c.name, "healthy: " + strconv.FormatBool(c.st.Healthy), "message: " + squashHTML(c.st.Message)}
	}
	return "", []string{"(empty)"}
}
