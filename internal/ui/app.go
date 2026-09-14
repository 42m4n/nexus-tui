package ui

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"nexus-tui/internal/nexus"
)

// Version is set at build time: -ldflags "-X ui.Version=vX.Y.Z"
var Version = "dev"

// bar modes for the k9s-style : command and / filter inputs.
const (
	barNone = iota
	barCmd
	barFilter
)

type Model struct {
	c         *nexus.Client
	clients   map[string]*nexus.Client
	profile   string
	profiles  []string
	status    string
	err       string
	msg       string
	msgExpiry time.Time

	// stack is the crumbs trail; top is the current resource view.
	stack []viewState
	// bar is the shared : / input; barMode tells which.
	bar     textinput.Model
	barMode int
	cmdHist []string
	histIdx int

	// searchVer cancels stale debounced searches.
	searchVer int

	// describe overlay content for the top vDescribe entry.
	descTitle string
	descLines []string

	showHelp bool
	loading  bool

	repos []nexus.Repository
	comps []nexus.Component
	tasks []nexus.Task
	users []nexus.User
	roles []nexus.Role
	privs []nexus.Privilege
	blobs []nexus.BlobStore

	checks   []checkRow
	readOnly nexus.ReadOnlyState
	roKnown  bool

	nxVer string // server product version, "" until loaded or failed

	query textinput.Model

	confirm confirmModal

	width, height int
}

type checkRow struct {
	name string
	st   nexus.CheckResult
}

type confirmModal struct {
	active bool
	prompt string
	expect string
	path   string
	input  string
}

func New(clients map[string]*nexus.Client, current string) Model {
	ti := textinput.New()
	ti.Placeholder = "name"
	ti.CharLimit = 128
	bar := textinput.New()
	bar.CharLimit = 128
	profiles := make([]string, 0, len(clients))
	for name := range clients {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	if current == "" && len(profiles) > 0 {
		current = profiles[0]
	}
	return Model{
		c:        clients[current],
		clients:  clients,
		profile:  current,
		profiles: profiles,
		stack:    []viewState{newView(vRepos, "repos")},
		bar:      bar,
		query:    ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadStatus(m.c), loadRepos(m.c), loadOverview(m.c))
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
type invalidateDone struct {
	err  error
	repo string
}
type taskRunDone struct {
	err  error
	name string
}
type readOnlyLoaded struct {
	ro  nexus.ReadOnlyState
	err error
}
type checksLoaded struct {
	checks []checkRow
	err    error
}
type clearAlertMsg struct{}
type searchDebounceMsg struct {
	ver   int
	query string
}
type versionLoaded struct{ version string }

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
func doInvalidateCache(c *nexus.Client, repo string) tea.Cmd {
	return func() tea.Msg { return invalidateDone{c.InvalidateCache(repo), repo} }
}
func doRunTask(c *nexus.Client, id, name string) tea.Cmd {
	return func() tea.Msg { return taskRunDone{c.RunTask(id), name} }
}
func loadReadOnly(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { ro, err := c.ReadOnly(); return readOnlyLoaded{ro, err} }
}
func loadVersion(c *nexus.Client) tea.Cmd {
	return func() tea.Msg { v, _ := c.ServerVersion(); return versionLoaded{v} } // ponytail: silent fallback to "nx:-" on failure
}

// loadOverview fires the header background loads: system checks, read-only
// state, blob stores, and server version. All tolerate errors; the header
// degrades to "n/a" instead of showing an error.
func loadOverview(c *nexus.Client) tea.Cmd {
	return tea.Batch(loadChecks(c), loadReadOnly(c), loadBlobs(c), loadVersion(c))
}
func loadChecks(c *nexus.Client) tea.Cmd {
	return func() tea.Msg {
		m, err := c.StatusChecks()
		if err != nil {
			return checksLoaded{err: err}
		}
		rows := make([]checkRow, 0, len(m))
		for name, st := range m {
			rows = append(rows, checkRow{name: name, st: st})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
		return checksLoaded{checks: rows}
	}
}

// enterHealth fires the health loads: system status checks,
// read-only state, and blob store space.
func (m Model) enterHealth() (tea.Model, tea.Cmd) {
	return m, tea.Batch(loadChecks(m.c), loadReadOnly(m.c), loadBlobs(m.c))
}

// ---- stack helpers ----

// top returns the current view, ensuring the stack is never empty.
func (m *Model) top() *viewState {
	if len(m.stack) == 0 {
		m.stack = append(m.stack, newView(vRepos, "repos"))
	}
	return &m.stack[len(m.stack)-1]
}

func (m *Model) push(v viewState) {
	m.stack = append(m.stack, v)
}

func (m *Model) pop() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
}

// openCommand pushes the view for kind, wiring its loader and initial filter.
// Repeating the current view refreshes instead of stacking duplicates.
func (m Model) openCommand(kind viewKind, arg, filter string) (tea.Model, tea.Cmd) {
	if top := m.top(); top.kind == kind && filter == "" {
		switch kind {
		case vComps:
			if arg == "" || arg == top.repo {
				return m.refresh()
			}
		case vSearch:
			if arg == "" {
				m.query.Focus()
				return m, textinput.Blink
			}
		default:
			if arg == "" {
				return m.refresh()
			}
		}
	}
	switch kind {
	case vRepos:
		vs := newView(vRepos, "repos")
		vs.filter = filter
		m.push(vs)
		return m, loadRepos(m.c)
	case vComps:
		repo := arg
		if repo == "" {
			repo = m.selectedRepoName()
		}
		if repo == "" {
			m.err = "usage: :comp <repository>"
			return m, nil
		}
		vs := newView(vComps, repo)
		vs.repo = repo
		vs.filter = filter
		m.push(vs)
		m.loading = true
		return m, loadComps(m.c, repo)
	case vSearch:
		vs := newView(vSearch, "search")
		vs.filter = filter
		m.push(vs)
		m.query.Focus()
		if arg != "" {
			m.query.SetValue(arg)
			m.loading = true
			return m, searchComps(m.c, arg)
		}
		return m, textinput.Blink
	case vTasks:
		vs := newView(vTasks, "tasks")
		vs.filter = filter
		m.push(vs)
		return m, loadTasks(m.c)
	case vUsers:
		vs := newView(vUsers, "users")
		vs.filter = filter
		m.push(vs)
		return m, loadUsers(m.c)
	case vRoles:
		vs := newView(vRoles, "roles")
		vs.filter = filter
		m.push(vs)
		return m, loadRoles(m.c)
	case vPrivs:
		vs := newView(vPrivs, "privileges")
		vs.filter = filter
		m.push(vs)
		return m, loadPrivs(m.c)
	case vBlobs:
		vs := newView(vBlobs, "blobstores")
		vs.filter = filter
		m.push(vs)
		return m, loadBlobs(m.c)
	case vHealth:
		vs := newView(vHealth, "health")
		vs.filter = filter
		m.push(vs)
		return m.enterHealth()
	case vCtx:
		vs := newView(vCtx, "contexts")
		vs.filter = filter
		m.push(vs)
		return m, nil
	}
	return m, nil
}

// selectedRepoName returns the selected repo on the repos view, if any.
func (m *Model) selectedRepoName() string {
	for i := len(m.stack) - 1; i >= 0; i-- {
		if m.stack[i].kind == vRepos && len(m.repos) > 0 {
			vs := &m.stack[i]
			if idx := m.selectedIndex(vs); idx >= 0 {
				return m.repos[idx].Name
			}
		}
		if m.stack[i].kind == vComps && m.stack[i].repo != "" {
			return m.stack[i].repo
		}
	}
	return ""
}

// ---- update ----

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := m.err + "|" + m.msg
	m2, cmd := m.update(msg)
	s := m2.(Model)
	if s.err+m.msg+s.msg != before && (s.err != "" || s.msg != "") {
		s.msgExpiry = time.Now().Add(4 * time.Second)
		if cmd == nil {
			cmd = tea.Tick(4*time.Second+50*time.Millisecond, func(time.Time) tea.Msg { return clearAlertMsg{} })
		}
	}
	return s, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.msg = ""
		}
		return m, nil
	case reposLoaded:
		m.repos = msg.repos
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
		}
		return m, nil
	case compsLoaded:
		m.comps = msg.comps
		m.loading = false
		if top := m.top(); top.kind == vComps || top.kind == vSearch {
			top.sel = 0
		}
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
		}
		return m, nil
	case tasksLoaded:
		m.tasks = msg.tasks
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
		}
		return m, nil
	case usersLoaded:
		m.users = msg.users
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
		}
		return m, nil
	case rolesLoaded:
		m.roles = msg.roles
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
		}
		return m, nil
	case privsLoaded:
		m.privs = msg.privs
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
		}
		return m, nil
	case blobsLoaded:
		m.blobs = msg.blobs
		if msg.err != nil && (m.top().kind == vBlobs || m.top().kind == vHealth) {
			m.err = msg.err.Error()
		} else if msg.err == nil {
			m.err = ""
		}
		return m, nil
	case actionDone:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.err = ""
			m.msg = ""
			m.confirm = confirmModal{}
			if m.top().kind == vRepos {
				return m, loadRepos(m.c)
			}
			if m.top().kind == vUsers {
				return m, loadUsers(m.c)
			}
		}
		return m, nil
	case invalidateDone:
		m.confirm = confirmModal{}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.msg = ""
		} else {
			m.err = ""
			m.msg = ""
			m.msg = "cache invalidated: " + msg.repo
		}
		return m, nil
	case taskRunDone:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.msg = ""
		} else {
			m.err = ""
			m.msg = "task started: " + msg.name
		}
		return m, nil
	case readOnlyLoaded:
		if msg.err == nil {
			m.readOnly = msg.ro
			m.roKnown = true
		} else if m.top().kind == vHealth {
			m.err = msg.err.Error() // only the health view consumes this directly
		}
		return m, nil
	case versionLoaded:
		m.nxVer = msg.version
		return m, nil
	case checksLoaded:
		if msg.err == nil {
			m.checks = msg.checks
		} else if m.top().kind == vHealth {
			m.err = msg.err.Error() // only the health view consumes this directly
		}
		return m, nil
	case clearAlertMsg:
		if !m.msgExpiry.IsZero() && !time.Now().Before(m.msgExpiry) {
			m.err, m.msg, m.msgExpiry = "", "", time.Time{}
		}
		return m, nil
	case searchDebounceMsg:
		if msg.ver == m.searchVer && msg.query != "" {
			m.loading = true
			return m, searchComps(m.c, msg.query)
		}
		return m, nil
	case tea.KeyMsg:
		if !m.msgExpiry.IsZero() && time.Since(m.msgExpiry) > 4*time.Second {
			m.err, m.msg, m.msgExpiry = "", "", time.Time{}
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		return m.handleConfirm(msg)
	}
	if m.barMode != barNone {
		return m.handleBar(msg)
	}
	if m.showHelp {
		switch msg.String() {
		case "esc", "?", "q":
			m.showHelp = false
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	// Search edit mode: printable keys go to the query input and trigger debounced search.
	if m.top().kind == vSearch && m.query.Focused() {
		switch msg.String() {
		case "enter":
			m.loading = true
			return m, searchComps(m.c, m.query.Value())
		case "esc":
			m.query.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		m.searchVer++
		ver, q := m.searchVer, m.query.Value()
		debounceCmd := tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
			return searchDebounceMsg{ver: ver, query: q}
		})
		return m, tea.Batch(cmd, debounceCmd)
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case ":":
		m.barMode = barCmd
		m.bar.SetValue("")
		m.bar.Focus()
		return m, textinput.Blink
	case "/":
		m.barMode = barFilter
		m.bar.SetValue(m.top().filter)
		m.bar.Focus()
		return m, textinput.Blink
	case "?":
		m.showHelp = true
		return m, nil
	case "ctrl+p":
		return m.openCommand(vCtx, "", "")
	case "esc":
		m.query.Blur()
		m.pop()
		return m, nil
	case "ctrl+r":
		return m.refresh()
	case "r":
		return m.refresh()
	case "t":
		return m.startRunTask()
	case "o":
		return m.cycleSort()
	case "O":
		if top := m.top(); top.kind != vDescribe {
			top.sortAsc = !top.sortAsc
		}
		return m, nil
	case "ctrl+d":
		return m.startDelete()
	case "d", "y":
		return m.openDescribe()
	case "i":
		return m.startInvalidate()
	case "enter":
		return m.activate()
	case "e":
		if m.top().kind == vSearch {
			m.query.Focus()
			return m, textinput.Blink
		}
		return m, nil
	// Legacy numeric shortcuts, kept working during the transition.
	case "1":
		return m.openCommand(vRepos, "", "")
	case "2":
		return m.openCommand(vSearch, "", "")
	case "3":
		return m.openCommand(vTasks, "", "")
	case "4":
		return m.openCommand(vUsers, "", "")
	case "5":
		return m.openCommand(vHealth, "", "")
	}

	moveCursor(msg, m.rowCount(), &m.top().sel)
	return m, nil
}

// rowCount is the display row count of the current table view.
func (m Model) rowCount() int {
	top := m.top()
	if top.kind == vDescribe {
		return len(m.descLines)
	}
	return len(m.displayOrder(top))
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	switch m.top().kind {
	case vRepos:
		return m, loadRepos(m.c)
	case vComps:
		if m.top().repo == "" {
			return m, nil
		}
		m.loading = true
		return m, loadComps(m.c, m.top().repo)
	case vSearch:
		m.loading = true
		return m, searchComps(m.c, m.query.Value())
	case vTasks:
		return m, loadTasks(m.c)
	case vUsers:
		return m, loadUsers(m.c)
	case vRoles:
		return m, loadRoles(m.c)
	case vPrivs:
		return m, loadPrivs(m.c)
	case vBlobs:
		return m, loadBlobs(m.c)
	case vHealth:
		return m.enterHealth()
	}
	return m, nil
}

// cycleSort moves to the next sortable column.
func (m Model) cycleSort() (tea.Model, tea.Cmd) {
	top := m.top()
	n := len(columns(top.kind))
	if n == 0 {
		return m, nil
	}
	if top.sortCol < 0 {
		top.sortCol = 0
		top.sortAsc = true
	} else if top.sortCol < n-1 {
		top.sortCol++
	} else {
		top.sortCol = -1 // back to server order
	}
	top.sel = clampSel(top.sel, len(m.displayOrder(top)))
	return m, nil
}

// activate is enter on the current row: drill in, switch, or describe.
func (m Model) activate() (tea.Model, tea.Cmd) {
	top := m.top()
	switch top.kind {
	case vRepos:
		if idx := m.selectedIndex(top); idx >= 0 {
			repo := m.repos[idx].Name
			vs := newView(vComps, repo)
			vs.repo = repo
			m.push(vs)
			m.loading = true
			return m, loadComps(m.c, repo)
		}
	case vCtx:
		if idx := m.selectedIndex(top); idx >= 0 {
			return m.switchProfile(m.profiles[idx])
		}
	case vDescribe:
		return m, nil
	default:
		return m.openDescribe()
	}
	return m, nil
}

// openDescribe pushes a detail view for the selected row.
func (m Model) openDescribe() (tea.Model, tea.Cmd) {
	top := m.top()
	if top.kind == vDescribe || top.kind == vCtx {
		return m, nil
	}
	title, lines := m.describe(top)
	if title == "" {
		m.err = "nothing to describe"
		return m, nil
	}
	vs := newView(vDescribe, title)
	m.push(vs)
	m.descTitle = title
	m.descLines = lines
	return m, nil
}

func (m Model) handleBar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.barMode == barFilter {
			m.top().filter = "" // cancel live filter
			m.top().sel = 0
		}
		m.barMode = barNone
		m.bar.Blur()
		m.histIdx = len(m.cmdHist)
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.bar.Value())
		mode := m.barMode
		m.barMode = barNone
		m.bar.Blur()
		if mode == barFilter {
			m.top().filter = val
			m.top().sel = 0
			return m, nil
		}
		if val == "" {
			return m, nil
		}
		m.cmdHist = append(m.cmdHist, val)
		m.histIdx = len(m.cmdHist)
		if val == "q" || val == "quit" || val == "exit" {
			return m, tea.Quit
		}
		kind, arg, filter, ok := resolveAlias(val)
		if !ok {
			m.err = "unknown command: " + val + "  (try :repos :tasks :users :health :ctx)"
			return m, nil
		}
		// :comp without arg drills into the selected repo, like enter.
		if kind == vComps && arg != "" && m.top().kind == vRepos {
			if idx := m.selectedIndex(m.top()); idx >= 0 && arg == m.repos[idx].Name {
				return m.activate()
			}
		}
		// :search <q> runs the query immediately.
		return m.openCommand(kind, arg, filter)
	case "tab":
		// Complete the :command word; keep any trailing args. The bar value
		// holds no leading ':' — the view renders that prefix.
		if m.barMode == barCmd {
			cmdStr := strings.TrimPrefix(m.bar.Value(), ":")
			words := strings.SplitN(cmdStr, " ", 2)
			if done := commandCompletion(words[0]); done != "" {
				if len(words) > 1 {
					m.bar.SetValue(done + " " + words[1])
				} else {
					m.bar.SetValue(done)
				}
				m.bar.CursorEnd()
			}
		}
		return m, nil
	case "up":
		if m.barMode == barCmd && len(m.cmdHist) > 0 {
			if m.histIdx > 0 {
				m.histIdx--
			}
			m.bar.SetValue(m.cmdHist[m.histIdx])
		}
		return m, nil
	case "down":
		if m.barMode == barCmd && len(m.cmdHist) > 0 {
			if m.histIdx < len(m.cmdHist)-1 {
				m.histIdx++
				m.bar.SetValue(m.cmdHist[m.histIdx])
			} else {
				m.histIdx = len(m.cmdHist)
				m.bar.SetValue("")
			}
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.bar, cmd = m.bar.Update(msg)
	if m.barMode == barFilter {
		m.top().filter = m.bar.Value()
		m.top().sel = 0
	}
	return m, cmd
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
	if top := m.top(); top.kind == vRepos && len(m.repos) > 0 {
		if idx := m.selectedIndex(top); idx >= 0 {
			r := m.repos[idx]
			m.confirm = confirmModal{active: true, prompt: "delete repository", expect: r.Name, path: "/repositories/" + r.Name}
			return m, nil
		}
	}
	if top := m.top(); top.kind == vUsers && len(m.users) > 0 {
		if idx := m.selectedIndex(top); idx >= 0 {
			u := m.users[idx]
			m.confirm = confirmModal{active: true, prompt: "delete user", expect: u.UserID, path: "/security/users/" + u.UserID}
			return m, nil
		}
	}
	m.err = "delete not available in this view (repos, users)"
	return m, nil
}

func (m Model) startInvalidate() (tea.Model, tea.Cmd) {
	if top := m.top(); top.kind == vRepos && len(m.repos) > 0 {
		if idx := m.selectedIndex(top); idx >= 0 {
			r := m.repos[idx]
			if r.Type == "hosted" {
				m.err = "hosted repositories have no cache"
				return m, nil
			}
			m.err = ""
			return m, doInvalidateCache(m.c, r.Name)
		}
	}
	m.err = "invalidate not available in this view (repos)"
	return m, nil
}

func (m Model) startRunTask() (tea.Model, tea.Cmd) {
	if !m.c.Writes {
		m.err = "writes disabled; restart with --allow-writes"
		return m, nil
	}
	if top := m.top(); top.kind == vTasks && len(m.tasks) > 0 {
		if idx := m.selectedIndex(top); idx >= 0 {
			t := m.tasks[idx]
			m.err = ""
			return m, doRunTask(m.c, t.ID, t.Name)
		}
	}
	m.err = "run not available in this view (tasks)"
	return m, nil
}

// switchProfile points the model at another instance, drops all cached
// state, and reloads the repos view.
func (m Model) switchProfile(name string) (tea.Model, tea.Cmd) {
	if name == m.profile {
		m.pop()
		return m, nil
	}
	m.c = m.clients[name]
	m.profile = name
	m.stack = []viewState{newView(vRepos, "repos")}
	m.status, m.err, m.msg = "", "", ""
	m.repos, m.comps = nil, nil
	m.tasks = nil
	m.users, m.roles, m.privs, m.blobs = nil, nil, nil, nil
	m.checks = nil
	m.readOnly, m.roKnown = nexus.ReadOnlyState{}, false
	m.nxVer = ""
	m.descLines, m.descTitle = nil, ""
	m.showHelp = false
	m.barMode = barNone
	m.query.Blur()
	m.query.SetValue("")
	return m, tea.Batch(loadStatus(m.c), loadRepos(m.c), loadOverview(m.c))
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
