package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"nexus-tui/internal/nexus"
)

type screen int

const (
	scrBrowse screen = iota
	scrSearch
	scrTasks
	scrAdmin
	scrAdminList
)

type Model struct {
	c      *nexus.Client
	status string
	err    string

	screen screen
	focus  int // 0 = left/top pane, 1 = right/detail pane

	repos    []nexus.Repository
	repoSel  int
	comps    []nexus.Component
	compSel  int
	loading  bool

	tasks   []nexus.Task
	taskSel int

	adminMenu []string
	adminSel  int
	adminKind string // users|roles|privileges|blobstores
	users     []nexus.User
	userSel   int
	roles     []nexus.Role
	roleSel   int
	privs     []nexus.Privilege
	privSel   int
	blobs     []nexus.BlobStore
	blobSel   int

	query textinput.Model

	confirm confirmModal

	width, height int
}

type confirmModal struct {
	active bool
	prompt string
	expect string
	path   string
	input  string
}

func New(c *nexus.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "name"
	ti.CharLimit = 128
	m := Model{
		c:         c,
		query:     ti,
		adminMenu: []string{"Users", "Roles", "Privileges", "Blob Stores"},
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadStatus(m.c), loadRepos(m.c))
}

// ---- messages ----

type statusLoaded struct {
	status string
	err    error
}
type reposLoaded struct {
	repos []nexus.Repository
	err   error
}
type compsLoaded struct {
	comps []nexus.Component
	err   error
}
type tasksLoaded struct {
	tasks []nexus.Task
	err   error
}
type usersLoaded struct {
	users []nexus.User
	err   error
}
type rolesLoaded struct {
	roles []nexus.Role
	err   error
}
type privsLoaded struct {
	privs []nexus.Privilege
	err   error
}
type blobsLoaded struct {
	blobs []nexus.BlobStore
	err   error
}
type actionDone struct{ err error }

// ---- commands ----

func loadStatus(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { s, err := c.Status(); return statusLoaded{s, err} }
}
func loadRepos(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { r, err := c.Repositories(); return reposLoaded{r, err} }
}
func loadComps(c *nexus.Client, repo string) tea.Cmd {
	return func() tea.Msg { cs, err := c.Components(repo); return compsLoaded{cs, err} }
}
func searchComps(c *nexus.Client, q string) tea.Cmd {
	return func() tea.Msg {
		cs, err := c.Search(map[string][]string{"name": {q}})
		return compsLoaded{cs, err}
	}
}
func loadTasks(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { t, err := c.Tasks(); return tasksLoaded{t, err} }
}
func loadUsers(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { u, err := c.Users(); return usersLoaded{u, err} }
}
func loadRoles(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { r, err := c.Roles(); return rolesLoaded{r, err} }
}
func loadPrivs(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { p, err := c.Privileges(); return privsLoaded{p, err} }
}
func loadBlobs(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { b, err := c.BlobStores(); return blobsLoaded{b, err} }
}
func doDelete(c *nexus.Client, path string) tea.Cmd {
	return func() tea.Msg { return actionDone{c.Delete(path)} }
}

func (m Model) adminLoad() tea.Cmd {
	switch m.adminKind {
	case "users":
		return loadUsers(m.c)
	case "roles":
		return loadRoles(m.c)
	case "privileges":
		return loadPrivs(m.c)
	case "blobstores":
		return loadBlobs(m.c)
	}
	return nil
}

// ---- update ----

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case statusLoaded:
		if msg.err != nil {
			m.status = "unreachable"
			m.err = msg.err.Error()
		} else {
			m.status = msg.status
			m.err = ""
		}
		return m, nil
	case reposLoaded:
		m.repos = msg.repos
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case compsLoaded:
		m.comps = msg.comps
		m.compSel = 0
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case tasksLoaded:
		m.tasks = msg.tasks
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case usersLoaded:
		m.users = msg.users
		m.userSel = 0
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case rolesLoaded:
		m.roles = msg.roles
		m.roleSel = 0
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case privsLoaded:
		m.privs = msg.privs
		m.privSel = 0
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case blobsLoaded:
		m.blobs = msg.blobs
		m.blobSel = 0
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
		}
		return m, nil
	case actionDone:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.confirm = confirmModal{}
			switch {
			case m.screen == scrBrowse:
				return m, loadRepos(m.c)
			case m.adminKind == "users":
				return m, loadUsers(m.c)
			}
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		return m.handleConfirm(msg)
	}
	k := msg.String()

	switch k {
	case "ctrl+c", "q":
		if m.screen == scrSearch && k == "q" {
			break
		}
		return m, tea.Quit
	case "esc":
		m.screen = scrBrowse
		m.query.Blur()
		return m, nil
	case "1":
		m.screen = scrBrowse
		m.query.Blur()
		return m, nil
	case "2":
		m.screen = scrSearch
		m.query.Focus()
		return m, textinput.Blink
	case "3":
		m.screen = scrTasks
		m.query.Blur()
		return m, loadTasks(m.c)
	case "4":
		m.screen = scrAdmin
		m.query.Blur()
		return m, nil
	case "r":
		switch m.screen {
		case scrBrowse:
			return m, loadRepos(m.c)
		case scrTasks:
			return m, loadTasks(m.c)
		case scrAdminList:
			return m, m.adminLoad()
		}
	case "d":
		return m.startDelete()
	case "tab":
		m.focus = 1 - m.focus
		return m, nil
	}

	switch m.screen {
	case scrBrowse:
		return m.browseKey(msg)
	case scrSearch:
		return m.searchKey(msg)
	case scrTasks:
		return m.listKey(msg, len(m.tasks), &m.taskSel)
	case scrAdmin:
		moveCursor(msg, len(m.adminMenu), &m.adminSel)
		if msg.String() == "enter" || msg.String() == "right" || msg.String() == "l" {
			kinds := []string{"users", "roles", "privileges", "blobstores"}
			m.adminKind = kinds[m.adminSel]
			m.screen = scrAdminList
			return m, m.adminLoad()
		}
		return m, nil
	case scrAdminList:
		return m.adminListKey(msg)
	}
	return m, nil
}

func (m Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.confirm = confirmModal{}
		return m, nil
	case "enter":
		if m.confirm.input != m.confirm.expect {
			m.err = "confirmation text did not match"
			return m, nil
		}
		return m, doDelete(m.c, m.confirm.path)
	case "backspace":
		if len(m.confirm.input) > 0 {
			m.confirm.input = m.confirm.input[:len(m.confirm.input)-1]
		}
		return m, nil
	default:
		r := msg.Runes
		if len(r) > 0 {
			m.confirm.input += string(r)
		}
		return m, nil
	}
}

func (m Model) startDelete() (tea.Model, tea.Cmd) {
	if !m.c.Writes {
		m.err = "writes disabled; restart with --allow-writes"
		return m, nil
	}
	if m.screen == scrBrowse && len(m.repos) > 0 {
		r := m.repos[m.repoSel]
		m.confirm = confirmModal{active: true, prompt: "delete repository", expect: r.Name, path: "/repositories/" + r.Name}
	}
	if m.screen == scrAdminList && m.adminKind == "users" && len(m.users) > 0 {
		u := m.users[m.userSel]
		m.confirm = confirmModal{active: true, prompt: "delete user", expect: u.UserID, path: "/security/users/" + u.UserID}
	}
	return m, nil
}

func (m Model) browseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == 0 {
		if cmd := moveCursor(msg, len(m.repos), &m.repoSel); cmd {
			return m, nil
		}
		if msg.String() == "enter" || msg.String() == "l" || msg.String() == "right" {
			if len(m.repos) > 0 {
				m.focus = 1
				m.loading = true
				return m, loadComps(m.c, m.repos[m.repoSel].Name)
			}
		}
		return m, nil
	}
	moveCursor(msg, len(m.comps), &m.compSel)
	if msg.String() == "h" || msg.String() == "left" {
		m.focus = 0
	}
	return m, nil
}

func (m Model) searchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		m.loading = true
		return m, searchComps(m.c, m.query.Value())
	}
	if msg.String() == "shift+tab" {
		return m, nil
	}
	var cmd tea.Cmd
	m.query, cmd = m.query.Update(msg)
	return m, cmd
}

func (m Model) listKey(msg tea.KeyMsg, n int, sel *int) (tea.Model, tea.Cmd) {
	moveCursor(msg, n, sel)
	return m, nil
}

func (m Model) adminListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.adminKind {
	case "users":
		moveCursor(msg, len(m.users), &m.userSel)
	case "roles":
		moveCursor(msg, len(m.roles), &m.roleSel)
	case "privileges":
		moveCursor(msg, len(m.privs), &m.privSel)
	case "blobstores":
		moveCursor(msg, len(m.blobs), &m.blobSel)
	}
	if msg.String() == "left" || msg.String() == "h" || msg.String() == "esc" {
		m.screen = scrAdmin
	}
	return m, nil
}

// moveCursor handles j/k/up/down; returns true if the list should consume the key.
func moveCursor(msg tea.KeyMsg, n int, sel *int) bool {
	if n == 0 {
		return false
	}
	switch msg.String() {
	case "up", "k":
		if *sel > 0 {
			*sel--
		}
		return true
	case "down", "j":
		if *sel < n-1 {
			*sel++
		}
		return true
	case "home", "g":
		*sel = 0
		return true
	case "end", "G":
		*sel = n - 1
		return true
	}
	return false
}

// ---- view ----

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selStyle    = lipgloss.NewStyle().Reverse(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	activePane  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("12"))
)

func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")

	bodyH := m.height - 4
	if bodyH < 3 {
		bodyH = 3
	}
	switch m.screen {
	case scrBrowse:
		b.WriteString(m.viewBrowse(bodyH))
	case scrSearch:
		b.WriteString(m.viewSearch(bodyH))
	case scrTasks:
		b.WriteString(m.viewTasks(bodyH))
	case scrAdmin:
		b.WriteString(m.viewAdmin(bodyH))
	case scrAdminList:
		b.WriteString(m.viewAdminList(bodyH))
	}
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m Model) header() string {
	dot := okStyle.Render("●")
	if m.status != "writable" {
		dot = errStyle.Render("●")
	}
	return fmt.Sprintf("%s Nexus TUI   status: %s   writes: %s", titleStyle.Render("NEXUS"), m.status, onOff(m.c.Writes)) + "   " + dot
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (m Model) footer() string {
	if m.err != "" {
		return errStyle.Render(trim(m.err, m.width))
	}
	if m.confirm.active {
		return fmt.Sprintf("%s %q -> type %q: [%s]  enter=confirm esc=cancel",
			errStyle.Render("CONFIRM"), m.confirm.prompt, m.confirm.expect, m.confirm.input)
	}
	switch m.screen {
	case scrBrowse:
		return "[j/k] move  [enter] open  [tab] pane  [d] delete repo  [1-4] views  [r] refresh  [q] quit"
	case scrSearch:
		return "type query  [enter] search  [esc] back  [2] focus"
	case scrTasks:
		return "[j/k] move  [r] refresh  [esc] back  [q] quit"
	case scrAdmin:
		return "[j/k] move  [enter] open  [esc] back"
	case scrAdminList:
		return "[j/k] move  [d] delete user  [r] refresh  [esc] back"
	}
	return ""
}

func (m Model) viewBrowse(h int) string {
	leftW := m.width/3 - 2
	if leftW < 20 {
		leftW = 20
	}
	rightW := m.width - leftW - 6
	if rightW < 20 {
		rightW = 20
	}
	rows := h - 2
	if rows < 1 {
		rows = 1
	}
	left := window(m.repos, m.repoSel, rows, func(i int) string {
		r := m.repos[i]
		return fmt.Sprintf("%s  %s/%s", r.Name, r.Format, r.Type)
	})
	right := window(m.comps, m.compSel, rows, func(i int) string {
		c := m.comps[i]
		return fmt.Sprintf("%s : %s : %s", c.Group, c.Name, c.Version)
	})
	lt, rt := "Repositories", "Components"
	if m.loading {
		rt += " (loading...)"
	}
	leftBox := pane(m.focus == 0, lt, left, leftW, h)
	rightBox := pane(m.focus == 1, rt, right, rightW, h)
	out := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	if m.focus == 1 && len(m.comps) > 0 {
		c := m.comps[m.compSel]
		out += "\n" + dimStyle.Render(fmt.Sprintf("assets: %d  id: %s", len(c.Assets), trim(c.ID, 40)))
	}
	return out
}

func (m Model) viewSearch(h int) string {
	rows := h - 3
	if rows < 1 {
		rows = 1
	}
	list := window(m.comps, m.compSel, rows, func(i int) string {
		c := m.comps[i]
		return fmt.Sprintf("%s : %s : %s  (%s)", c.Group, c.Name, c.Version, c.Repository)
	})
	return m.query.View() + "\n" + list
}

func (m Model) viewTasks(h int) string {
	rows := h - 2
	if rows < 1 {
		rows = 1
	}
	list := window(m.tasks, m.taskSel, rows, func(i int) string {
		t := m.tasks[i]
		return fmt.Sprintf("%-30s %-12s %s", t.Name, t.CurrentState, t.Type)
	})
	out := list
	if len(m.tasks) > 0 {
		t := m.tasks[m.taskSel]
		out += "\n" + dimStyle.Render(fmt.Sprintf("last run: %s  result: %s  next: %s",
			orDash(t.LastRun), orDash(t.LastRunResult), orDash(t.NextRun)))
	}
	return out
}

func (m Model) viewAdmin(h int) string {
	return window(m.adminMenu, m.adminSel, h-2, func(i int) string { return m.adminMenu[i] })
}

func (m Model) viewAdminList(h int) string {
	rows := h - 2
	if rows < 1 {
		rows = 1
	}
	switch m.adminKind {
	case "users":
		return window(m.users, m.userSel, rows, func(i int) string {
			u := m.users[i]
			return fmt.Sprintf("%-20s %-30s %s", u.UserID, u.Email, u.Source)
		})
	case "roles":
		return window(m.roles, m.roleSel, rows, func(i int) string {
			r := m.roles[i]
			return fmt.Sprintf("%-25s %s", r.Name, trim(r.Description, 60))
		})
	case "privileges":
		return window(m.privs, m.privSel, rows, func(i int) string {
			p := m.privs[i]
			return fmt.Sprintf("%-30s %-20s %s", p.Name, p.Type, trim(p.Description, 50))
		})
	case "blobstores":
		return window(m.blobs, m.blobSel, rows, func(i int) string {
			b := m.blobs[i]
			return fmt.Sprintf("%-20s %-8s blobs:%d size:%d", b.Name, b.Type, b.BlobCount, b.TotalSize)
		})
	}
	return ""
}

// window renders a scrollable, selectable list of n rows.
// ponytail: full-slice render with a simple offset; fine for thousands of rows, paginate if lists get huge.
func window[T any](items []T, sel, rows int, render func(int) string) string {
	if len(items) == 0 {
		return dimStyle.Render("(empty)")
	}
	if rows > len(items) {
		rows = len(items)
	}
	start := 0
	if sel >= rows {
		start = sel - rows + 1
	}
	if start+rows > len(items) {
		start = len(items) - rows
	}
	var b strings.Builder
	for i := start; i < start+rows && i < len(items); i++ {
		line := render(i)
		if i == sel {
			line = selStyle.Render(line)
		}
		b.WriteString(line)
		if i < start+rows-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func pane(active bool, title, body string, w, h int) string {
	st := paneStyle
	if active {
		st = activePane
	}
	content := titleStyle.Render(title) + "\n" + body
	return st.Width(w).Height(h - 2).Render(content)
}

func trim(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
