package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selStyle    = lipgloss.NewStyle().Reverse(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	headStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	tableBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
)

func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString("\n")
	b.WriteString(m.crumbs())
	b.WriteString("\n")

	bodyH := m.height - 6 // room for header, crumbs, 2-line footer
	if m.barMode != barNone {
		bodyH--
	}
	if bodyH < 3 {
		bodyH = 3
	}
	if m.showHelp {
		b.WriteString(m.viewHelp(bodyH))
	} else {
		b.WriteString(m.viewBody(bodyH))
	}
	b.WriteString("\n")
	if m.barMode != barNone {
		b.WriteString(m.viewBar())
		b.WriteString("\n")
	}
	b.WriteString(m.footer())
	return b.String()
}

// header renders the one-line instance banner. On narrow screens the right
// side drops the storage segment first, then writes, so profile, server
// version, and health stay visible as long as possible.
func (m Model) header() string {
	ver := strings.TrimPrefix(Version, "v")
	left := titleStyle.Render("NEXUS") + " " + dimStyle.Render(ver) + "  " + m.profile
	if host := m.c.Host(); host != "" {
		left += dimStyle.Render(" (" + host + ")")
	}
	nx := "nx:" + orDash(m.nxVer)
	if m.nxVer == "" {
		nx = dimStyle.Render(nx)
	}
	left += "  " + nx

	// health: x/y checks ok; dot red if any check failed
	dot := okStyle.Render("●")
	st := orDash(m.status)
	if m.status != "" && m.status != "writable" {
		dot = errStyle.Render("●")
	}
	if len(m.checks) > 0 {
		ok := 0
		for _, c := range m.checks {
			if c.st.Healthy {
				ok++
			}
		}
		chk := fmt.Sprintf("checks:%d/%d", ok, len(m.checks))
		if ok == len(m.checks) {
			st += " " + okStyle.Render(chk)
		} else {
			st += " " + errStyle.Render(chk)
			dot = errStyle.Render("●")
		}
	}
	if m.roKnown && m.readOnly.Frozen {
		ro := "RO:frozen"
		if r := m.readOnly.SummaryReason; r != "" {
			ro += "(" + trim(squashHTML(r), 20) + ")"
		}
		st += " " + errStyle.Render(ro)
		dot = errStyle.Render("●")
	}

	// storage: ok blobs / total free space
	var segStorage string
	if len(m.blobs) > 0 {
		okB := 0
		var free int64
		for _, b := range m.blobs {
			if !b.Unavailable && b.AvailableSpace >= 1<<30 {
				okB++
			}
			// File stores share a disk; AvailableSpace is the same
			// free-space value per disk — summing double counts it.
			// Take the max so the header matches the per-store rows.
			if !b.Unavailable && b.AvailableSpace > free {
				free = b.AvailableSpace
			}
		}
		// ponytail: max(AvailableSpace) across File stores; upgrade to sum only Group stores if multi-disk deployments matter
		mk := okStyle.Render
		if okB < len(m.blobs) {
			mk = errStyle.Render
		}
		segStorage = mk(fmt.Sprintf("blobs:%d/%d %s free", okB, len(m.blobs), humanBytes(free)))
	}

	writes := dimStyle.Render("writes:off")
	if m.c.Writes {
		writes = okStyle.Render("writes:on")
	}
	right := fmt.Sprintf("%s %s", dot, st)
	if segStorage != "" {
		right += "  " + segStorage
	}
	right += "  " + writes
	if m.loading {
		right = dimStyle.Render("(loading...)") + "  " + right
	}

	// narrow screens: drop segments from the right, storage first, so
	// health and writes stay visible as long as possible
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 && segStorage != "" {
		right = fmt.Sprintf("%s %s  %s", dot, st, writes)
		gap = m.width - lipgloss.Width(left) - lipgloss.Width(right)
	}
	if gap < 1 {
		right = trim(fmt.Sprintf("%s %s", dot, st), max(1, m.width-lipgloss.Width(left)))
		gap = m.width - lipgloss.Width(left) - lipgloss.Width(right)
	}
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) crumbs() string {
	parts := make([]string, 0, len(m.stack))
	for i, vs := range m.stack {
		t := vs.title
		if t == "" {
			t = vs.kind.defaultTitle()
		}
		if i == len(m.stack)-1 {
			parts = append(parts, titleStyle.Render(t))
		} else {
			parts = append(parts, dimStyle.Render(t))
		}
	}
	s := strings.Join(parts, dimStyle.Render(" > "))
	top := m.top()
	if n := m.rowCount(); n > 0 && top.kind != vDescribe {
		s += dimStyle.Render(fmt.Sprintf(" (%d)", n))
	}
	if f := top.filter; f != "" {
		s += dimStyle.Render("  /" + f)
	}
	return trim(s, max(1, m.width))
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (m Model) footer() string {
	var status string
	switch {
	case m.confirm.active:
		status = fmt.Sprintf("%s %q -> type %q: [%s]  enter=confirm esc=cancel",
			errStyle.Render("CONFIRM"), m.confirm.prompt, m.confirm.expect, m.confirm.input)
	case m.err != "":
		status = errStyle.Render(trim(m.err, m.width))
	case m.msg != "":
		status = okStyle.Render(trim(m.msg, m.width))
	}

	keys := "[: cmd] [/ filter] [o sort] [d describe] [enter open] [esc back] [? help] [q quit]"
	switch m.top().kind {
	case vRepos:
		keys = "[i inval-cache] [ctrl-d delete] " + keys
	case vSearch:
		if m.query.Focused() {
			keys = "type to search live  [esc] stop editing"
		} else {
			keys = "[e edit query] [enter describe] " + keys
		}
	case vHealth:
		keys = "[d describe] [r refresh] " + keys
	case vCtx:
		keys = "[enter switch] [esc back] [q quit]"
	case vDescribe:
		keys = "[j/k scroll] [esc back] [q quit]"
	case vUsers:
		keys = "[ctrl-d delete] " + keys
	}
	keyLine := dimStyle.Render(trim(keys, max(1, m.width)))

	if status != "" {
		return status + "\n" + keyLine
	}
	return "\n" + keyLine
}

func (m Model) viewBar() string {
	if m.barMode == barCmd {
		return ":" + m.bar.View()
	}
	return "/" + m.bar.View()
}

func (m Model) viewBody(h int) string {
	top := m.top()
	switch top.kind {
	case vSearch:
		return m.viewSearch(h)
	case vHealth:
		return m.viewHealth(h)
	case vDescribe:
		rows := max(1, h-3)
		body := window(m.descLines, top.sel, rows, func(i int) string { return m.descLines[i] })
		return tableBorder.Width(m.width - 2).Render(
			titleStyle.Render(trim(m.descTitle, max(1, m.width-8))) + "\n" + body)
	default:
		return m.viewTable(top, h)
	}
}

// viewTable renders a bordered, aligned table for a resource view.
func (m Model) viewTable(vs *viewState, h int) string {
	rows := m.rows(vs.kind)
	order := filterSort(rows, vs.filter, vs.sortCol, vs.sortAsc)
	vs.sel = clampSel(vs.sel, len(order))
	if len(order) == 0 {
		if vs.filter != "" {
			return dimStyle.Render("(no match for /" + vs.filter + ")")
		}
		return dimStyle.Render("(empty)")
	}
	return m.renderBlock(vs, rows, order, max(1, h-4))
}

// renderBlock builds header + separator + rows and wraps them in a border.
// vis counts body rows; header, separator and border are overhead.
func (m Model) renderBlock(vs *viewState, rows [][]string, order []int, vis int) string {
	headers := columns(vs.kind)
	inner := max(20, m.width-6)
	widths := colWidths(headers, rows, inner)
	var b strings.Builder
	b.WriteString(renderHeader(headers, widths, vs.sortCol, vs.sortAsc))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", inner)))
	b.WriteString("\n")
	b.WriteString(window(order, vs.sel, vis, func(i int) string {
		row := rows[order[i]]
		cells := make([]string, len(row))
		for j, c := range row {
			cells[j] = colorCell(vs.kind, j, c)
		}
		return padCells(cells, widths)
	}))
	return tableBorder.Width(m.width - 2).Render(b.String())
}

// colorCell paints one cell before padding.
func colorCell(kind viewKind, col int, cell string) string {
	switch kind {
	case vHealth:
		if col == 1 {
			if cell == "OK" {
				return okStyle.Render(cell)
			}
			return errStyle.Render(cell)
		}
	case vTasks:
		if col == 1 {
			switch cell {
			case "WAITING":
				return warnStyle.Render(cell)
			case "FAILED", "ERROR":
				return errStyle.Render(cell)
			}
		}
	}
	return cell
}

func renderHeader(headers []string, widths []int, sortCol int, asc bool) string {
	out := make([]string, len(headers))
	for i, hc := range headers {
		w := hc
		if i < len(widths) {
			w = fitCell(hc, widths[i])
			w += strings.Repeat(" ", max(0, widths[i]-lipgloss.Width(w)))
		}
		if i == sortCol {
			arrow := "▲"
			if !asc {
				arrow = "▼"
			}
			out[i] = headStyle.Render(w + arrow)
		} else {
			out[i] = headStyle.Render(w)
		}
	}
	return strings.Join(out, "  ")
}

func (m Model) viewSearch(h int) string {
	hint := dimStyle.Render("query: type to search live (esc stops editing)")
	if m.query.Focused() {
		hint = dimStyle.Render("query: searching as you type...")
	}
	top := m.top()
	return m.query.View() + "\n" + hint + "\n" + m.viewTable(top, h-2)
}

// viewHealth renders: server line, checks table, storage summary.
func (m Model) viewHealth(h int) string {
	var b strings.Builder
	ro := "unknown"
	if m.roKnown {
		ro = onOff(m.readOnly.Frozen)
	}
	writes := okStyle.Render("writable")
	if m.status != "writable" {
		writes = errStyle.Render(m.status)
	}
	roField := "read-only: " + ro
	if m.roKnown && m.readOnly.Frozen {
		if m.readOnly.SummaryReason != "" {
			roField += " (" + m.readOnly.SummaryReason + ")"
		}
		roField = errStyle.Render(roField)
	}
	b.WriteString(fmt.Sprintf("server: %s   %s", writes, roField))

	top := m.top()
	rows := m.rows(vHealth)
	order := filterSort(rows, top.filter, top.sortCol, top.sortAsc)
	top.sel = clampSel(top.sel, len(order))
	vis := max(1, h-6-len(m.blobs)-2)
	if len(order) == 0 {
		b.WriteString("\n" + dimStyle.Render("(empty)"))
	} else {
		inner := max(20, m.width-6)
		widths := colWidths(columns(vHealth), rows, inner-4) // room for the ● marker
		var t strings.Builder
		t.WriteString(renderHeader(columns(vHealth), widths, top.sortCol, top.sortAsc))
		t.WriteString("\n")
		t.WriteString(dimStyle.Render(strings.Repeat("─", inner)))
		t.WriteString("\n")
		t.WriteString(window(order, top.sel, vis, func(i int) string {
			r := rows[order[i]]
			marker := okStyle.Render("●")
			if r[1] != "OK" {
				marker = errStyle.Render("●")
			}
			cells := make([]string, len(r))
			for j, c := range r {
				cells[j] = colorCell(vHealth, j, c)
			}
			return marker + "  " + padCells(cells, widths)
		}))
		b.WriteString("\n" + tableBorder.Width(m.width-2).Render(t.String()))
	}

	b.WriteString("\n" + titleStyle.Render("storage"))
	for _, bl := range m.blobs {
		marker := okStyle.Render("●")
		if bl.Unavailable || bl.AvailableSpace < 1<<30 { // unavailable or under 1 GiB
			marker = errStyle.Render("●")
		}
		b.WriteString("\n" + fmt.Sprintf("%s %-20s %s free", marker, bl.Name, humanBytes(bl.AvailableSpace)))
	}
	return b.String()
}

// viewHelp renders a bordered two-column overlay: keys left, commands right.
func (m Model) viewHelp(bodyH int) string {
	keys := [][2]string{
		{":", "open command bar (tab completes)"},
		{"/", "filter rows live"},
		{"enter", "open / drill in / switch"},
		{"d / y", "describe selected row"},
		{"o / O", "sort column / reverse"},
		{"r", "refresh current view"},
		{"i", "invalidate cache (proxy/group)"},
		{"ctrl-d", "delete (repos, users; confirm)"},
		{"e", "edit search query"},
		{"ctrl+p", "switch server profile"},
		{"esc", "back / close overlay"},
		{"q", "quit"},
	}
	cmds := [][2]string{
		{":repos", "list repositories"},
		{":comp <repo>", "list components in repo"},
		{":search <q>", "search artifacts live"},
		{":tasks", "show scheduled tasks"},
		{":users", "list users"},
		{":roles", "list roles"},
		{":privs", "list privileges"},
		{":blobs", "list blob stores"},
		{":health", "system checks and storage"},
		{":ctx", "switch server profile"},
		{":q", "quit"},
	}
	keyW := 0
	for _, kv := range keys {
		if len(kv[0]) > keyW {
			keyW = len(kv[0])
		}
	}
	cmdW := 0
	for _, cv := range cmds {
		if len(cv[0]) > cmdW {
			cmdW = len(cv[0])
		}
	}

	inner := max(40, m.width-8)
	colW := (inner - 6) / 2
	keyLines := make([]string, 0, len(keys)+1)
	keyLines = append(keyLines, titleStyle.Render("Keys"))
	for _, kv := range keys {
		keyLines = append(keyLines, trim(fmt.Sprintf("  %-*s  %s", keyW, kv[0], kv[1]), colW))
	}
	cmdLines := make([]string, 0, len(cmds)+1)
	cmdLines = append(cmdLines, titleStyle.Render("Commands"))
	for _, cv := range cmds {
		cmdLines = append(cmdLines, trim(fmt.Sprintf("  %-*s  %s", cmdW, cv[0], cv[1]), colW))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Join(keyLines, "\n"), strings.Repeat(" ", 3), strings.Join(cmdLines, "\n"))
	return tableBorder.Width(m.width - 2).Render(body)
}

// squashHTML flattens embedded HTML tags and collapse whitespace; Nexus
// check messages contain <br>/<b> markup.
func squashHTML(s string) string {
	s = strings.ReplaceAll(s, "<br>", " ")
	s = strings.ReplaceAll(s, "<b>", " ")
	s = strings.ReplaceAll(s, "</b>", " ")
	return strings.Join(strings.Fields(s), " ")
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
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
