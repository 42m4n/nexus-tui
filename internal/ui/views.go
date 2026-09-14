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

	bodyH := m.height - 5
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

func (m Model) header() string {
	dot := okStyle.Render("●")
	if m.status != "writable" {
		dot = errStyle.Render("●")
	}
	st := m.status
	if st == "" {
		st = "…"
	}
	top := m.top()
	title := top.title
	if m.loading && (top.kind == vComps || top.kind == vSearch) {
		title += " (loading...)"
	}
	return fmt.Sprintf("%s Nexus %s %s %s %s",
		titleStyle.Render("NEXUS"), m.profile, dot+st,
		"writes:"+onOff(m.c.Writes), titleStyle.Render(trim(title, 30)))
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
	if f := m.top().filter; f != "" {
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
	if m.err != "" {
		return errStyle.Render(trim(m.err, m.width))
	}
	if m.confirm.active {
		return fmt.Sprintf("%s %q -> type %q: [%s]  enter=confirm esc=cancel",
			errStyle.Render("CONFIRM"), m.confirm.prompt, m.confirm.expect, m.confirm.input)
	}
	if m.msg != "" {
		return okStyle.Render(trim(m.msg, m.width))
	}
	keys := "[: cmd] [/ filter] [o sort] [d describe] [enter open] [esc back] [? help] [q quit]"
	switch m.top().kind {
	case vRepos:
		keys = "[enter open] [d describe] [i inval-cache] [ctrl-d delete] " + keys
	case vSearch:
		if m.query.Focused() {
			return "[enter] search  [esc] stop editing"
		}
		keys = "[e edit query] [enter describe] " + keys
	case vHealth:
		keys = "[d describe] [r refresh] " + keys
	case vCtx:
		keys = "[enter switch] [esc back] [q quit]"
	case vDescribe:
		keys = "[j/k scroll] [esc back] [q quit]"
	case vUsers:
		keys = "[d describe] [ctrl-d delete] " + keys
	}
	return dimStyle.Render(trim(keys, max(1, m.width)))
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
	hint := dimStyle.Render("query: (e to edit, enter in edit mode searches)")
	if m.query.Focused() {
		hint = dimStyle.Render("editing query — enter searches, esc stops")
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

func (m Model) viewHelp(bodyH int) string {
	_ = bodyH
	lines := []string{
		titleStyle.Render("Keys"),
		"  :            command (:repos :comp <r> :search :tasks :users :roles :privs :blobs :health :ctx)",
		"  /            filter current view (regex), enter applies, esc clears bar",
		"  enter        open (repos→components, ctx→switch) / describe row",
		"  d / y        describe selected row",
		"  o / O        cycle sort column / toggle direction",
		"  r            refresh current view",
		"  i            invalidate cache (proxy/group repo)",
		"  ctrl-d       delete (repos, users; typed confirm, needs --allow-writes)",
		"  e            edit search query (in search view)",
		"  esc          back (pop crumbs)   ? help   q quit   ctrl+p contexts",
		"",
		titleStyle.Render("Commands"),
		"  :repos | :comp <repo> | :search <q> | :tasks | :users | :roles | :privs | :blobs | :health | :ctx",
		"  append /<filter> to pre-filter, e.g. :repos /maven",
		"  :q quits. up/down recalls command history in : mode.",
	}
	return strings.Join(lines, "\n")
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
