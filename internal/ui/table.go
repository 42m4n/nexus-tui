package ui

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// matchFilter reports whether any cell matches f. Empty matches all.
// Case-insensitive regex; falls back to literal contains on bad regex.
func matchFilter(cells []string, f string) bool {
	if f == "" {
		return true
	}
	if re, err := regexp.Compile("(?i)" + f); err == nil {
		for _, c := range cells {
			if re.MatchString(c) {
				return true
			}
		}
		return false
	}
	fl := strings.ToLower(f)
	for _, c := range cells {
		if strings.Contains(strings.ToLower(c), fl) {
			return true
		}
	}
	return false
}

// filterSort returns display-order indices into rows after filter+sort.
// col < 0 disables sorting. Numeric-aware: all-digit columns sort as numbers.
func filterSort(rows [][]string, filter string, col int, asc bool) []int {
	idx := make([]int, 0, len(rows))
	for i, r := range rows {
		if matchFilter(r, filter) {
			idx = append(idx, i)
		}
	}
	if col < 0 || len(idx) < 2 {
		return idx
	}
	cell := func(i int) string {
		if col < len(rows[i]) {
			return rows[i][col]
		}
		return ""
	}
	sort.SliceStable(idx, func(a, b int) bool {
		va, vb := cell(idx[a]), cell(idx[b])
		if na, err1 := strconv.ParseInt(strings.TrimSpace(va), 10, 64); err1 == nil {
			if nb, err2 := strconv.ParseInt(strings.TrimSpace(vb), 10, 64); err2 == nil {
				if asc {
					return na < nb
				}
				return na > nb
			}
		}
		if asc {
			return strings.ToLower(va) < strings.ToLower(vb)
		}
		return strings.ToLower(va) > strings.ToLower(vb)
	})
	return idx
}

// clampSel keeps sel inside [0, n-1]; returns 0 when empty.
func clampSel(sel, n int) int {
	if n <= 0 {
		return 0
	}
	if sel < 0 {
		return 0
	}
	if sel >= n {
		return n - 1
	}
	return sel
}

// colWidths fits every column to its widest cell, shrinking the widest
// columns first so the joined line fits maxTotal.
func colWidths(headers []string, rows [][]string, maxTotal int) []int {
	n := len(headers)
	w := make([]int, n)
	for i, h := range headers {
		w[i] = lipgloss.Width(h)
	}
	for _, r := range rows {
		for i := 0; i < n && i < len(r); i++ {
			if c := lipgloss.Width(r[i]); c > w[i] {
				w[i] = c
			}
		}
	}
	for totalW(w)+2*(n-1) > maxTotal {
		bi := -1
		for i := range w {
			if w[i] > lipgloss.Width(headers[i]) && (bi < 0 || w[i] > w[bi]) {
				bi = i
			}
		}
		if bi < 0 {
			break
		}
		w[bi]--
	}
	return w
}

func totalW(w []int) int {
	t := 0
	for _, v := range w {
		t += v
	}
	return t
}

// padCells aligns cells to widths (numbers right, text left) and joins them.
func padCells(cells []string, widths []int) string {
	out := make([]string, len(widths))
	for i, w := range widths {
		s := ""
		if i < len(cells) {
			s = fitCell(cells[i], w)
		}
		if _, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			out[i] = strings.Repeat(" ", max(0, w-lipgloss.Width(s))) + s
		} else {
			out[i] = s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
		}
	}
	return strings.Join(out, "  ")
}

// fitCell truncates s to w cells (ASCII data; headers stay intact via widths).
func fitCell(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if len(r) > w {
		r = r[:w]
	}
	return string(r)
}
