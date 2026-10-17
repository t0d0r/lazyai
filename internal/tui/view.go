package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"lazyai/internal/agent"
	"lazyai/internal/display"
	"lazyai/internal/watch"
)

var (
	focusColor = lipgloss.Color("2") // green, as in lazydocker
	dimColor   = lipgloss.Color("8")

	titleStyle  = lipgloss.NewStyle().Bold(true)
	activeStyle = lipgloss.NewStyle().Bold(true).Foreground(focusColor)
	selStyle    = lipgloss.NewStyle().Reverse(true)
	selDimStyle = lipgloss.NewStyle().Bold(true)
	busyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	flagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle    = lipgloss.NewStyle().Foreground(dimColor)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	barStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	keyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
)

// minEvenListH is the smallest list panel (borders plus 3 rows) for which
// the column is split evenly.
const minEvenListH = 5

// layout is where each panel sits. Mouse handling uses it too.
type layout struct {
	leftW, rightW, bodyH int
	statusH              int
	listY, listH         [numLists]int
}

func (m model) layout() layout {
	l := layout{bodyH: max(m.height-1, 6), statusH: 3}
	l.leftW = max(min(m.width/3, 56), min(32, m.width/2))
	l.rightW = max(m.width-l.leftW, 10)
	avail := l.bodyH - l.statusH
	if share := avail / numLists; share >= minEvenListH {
		// Split evenly, as lazydocker does; leftover rows go to the top.
		for i := range numLists {
			l.listH[i] = share
			if i < avail%numLists {
				l.listH[i]++
			}
		}
	} else {
		// Too short to split: the panel in use grows, the others show one row.
		for i := range numLists {
			l.listH[i] = 3
		}
		l.listH[m.lastList] = max(avail-3*(numLists-1), 3)
	}
	y := l.statusH
	for i := range numLists {
		l.listY[i] = y
		y += l.listH[i]
	}
	return l
}

// listTop is the first row shown in list i, keeping the cursor visible.
func (m model) listTop(i int, l layout) int {
	visible := max(l.listH[i]-2, 1)
	return max(m.lists[i].cursor-visible+1, 0)
}

func (m model) View() string {
	l := m.layout()
	var left []string
	left = append(left, m.statusPanel(l))
	for i := range numLists {
		left = append(left, m.listPanel(i, l))
	}
	leftCol := cropLines(strings.Join(left, "\n"), l.bodyH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, m.mainPanel(l))
	body = cropLines(body, l.bodyH)

	switch m.popup {
	case popupHelp:
		body = center(body, m.helpPopup(), m.width, l.bodyH)
	case popupMenu:
		body = center(body, m.menuPopup(), m.width, l.bodyH)
	case popupConfirm:
		body = center(body, m.confirmPopup(), m.width, l.bodyH)
	case popupInput:
		body = center(body, m.inputPopup(), m.width, l.bodyH)
	}
	return body + "\n" + m.footer()
}

func (m model) statusPanel(l layout) string {
	busy, total := 0, 0
	for _, a := range m.all {
		if a.Role == agent.RoleHelper && !m.showHelpers {
			continue
		}
		total++
		if a.Status == "busy" {
			busy++
		}
	}
	line := fmt.Sprintf("%d running, %d busy", total, busy)
	right := "loading ..."
	if !m.updated.IsZero() {
		right = m.updated.Format("15:04:05")
	}
	return box(titleStyle.Render("lazyai"), []string{spread(line, dimStyle.Render(right), l.leftW-2)},
		l.leftW, l.statusH, false, "")
}

func (m model) listPanel(i int, l layout) string {
	lst := m.lists[i]
	focused := m.focus == i
	inner := l.leftW - 2
	now := time.Now()
	visible := max(l.listH[i]-2, 0)
	top := m.listTop(i, l)
	var lines []string
	if len(lst.rows) == 0 {
		lines = append(lines, dimStyle.Render("none"))
	}
	for r := top; r < len(lst.rows) && r < top+visible; r++ {
		a := lst.rows[r]
		t, _ := m.watcher.Tracker(a.PID)
		name := string(a.Kind) + " " + label(a)
		if i != listAgents {
			name = string(a.Kind) + " " + string(a.Role) + " " + label(a)
		}
		if a.Protected {
			name += " *"
		}
		right := fmt.Sprintf("%s %5.1f%%", a.Status, a.TreeCPU)
		if i == listModels {
			right += " " + display.Bytes(a.TreeRSS)
		}
		row := spread(name, right, inner)
		flagged := false
		for _, f := range watch.Flags(a, t, now) {
			flagged = flagged || f != "self"
		}
		switch {
		case r == lst.cursor && focused:
			row = selStyle.Render(row)
		case r == lst.cursor:
			row = selDimStyle.Render(row)
		case flagged:
			row = flagStyle.Render(row)
		case a.Status == "busy":
			row = busyStyle.Render(row)
		}
		lines = append(lines, row)
	}
	pos := ""
	if len(lst.rows) > 0 {
		pos = fmt.Sprintf("%d of %d", lst.cursor+1, len(lst.rows))
	}
	title := titleStyle.Render(fmt.Sprintf("%d %s", i+1, listTitles[i]))
	if focused {
		title = activeStyle.Render(fmt.Sprintf("%d %s", i+1, listTitles[i]))
	}
	return box(title, lines, l.leftW, l.listH[i], focused, pos)
}

// mainSize is the number of content lines and how many fit.
func (m model) mainSize() (int, int) {
	l := m.layout()
	return len(m.mainLines(l.rightW - 2)), max(l.bodyH-2, 1)
}

func (m model) mainPanel(l layout) string {
	inner := l.rightW - 2
	lines := m.mainLines(inner)
	visible := max(l.bodyH-2, 1)
	maxScroll := max(len(lines)-visible, 0)
	off := min(m.scroll, maxScroll)
	if m.follow && m.tab == tabScreen {
		off = maxScroll
	}
	lines = lines[off:]

	var tabs []string
	for i, t := range tabTitles {
		if i == m.tab {
			tabs = append(tabs, activeStyle.Render(t))
		} else {
			tabs = append(tabs, t)
		}
	}
	pos := ""
	if maxScroll > 0 {
		pos = fmt.Sprintf("%d-%d of %d", off+1, min(off+visible, off+len(lines)), off+len(lines))
	}
	return box(strings.Join(tabs, " - "), lines, l.rightW, l.bodyH, m.focus == focusMain, pos)
}

func (m model) mainLines(w int) []string {
	a, ok := m.selected()
	if !ok {
		return []string{dimStyle.Render("no " + strings.ToLower(listTitles[m.lastList]) + " running")}
	}
	switch m.tab {
	case tabScreen:
		return m.screenLines(a)
	case tabProcesses:
		return processLines(a)
	case tabStats:
		t, _ := m.watcher.Tracker(a.PID)
		return statsLines(a, t, w, max(m.layout().bodyH-2, 4))
	}
	return m.infoLines(a)
}

func (m model) infoLines(a agent.Agent) []string {
	now := time.Now()
	t, _ := m.watcher.Tracker(a.PID)
	fields := append([]display.Field{{Name: "state", Value: display.Status(a, t, now)}}, display.Details(a)...)
	fields = append(fields,
		display.Field{Name: "resources", Value: fmt.Sprintf("%.1f%% CPU, %s, up %s",
			a.CPU, display.Bytes(a.RSS), display.Duration(now.Sub(a.Start)))},
		display.Field{Name: "tree", Value: fmt.Sprintf("%.1f%% CPU, %s, %d child processes",
			a.TreeCPU, display.Bytes(a.TreeRSS), len(a.Children))},
		display.Field{Name: "flags", Value: display.Flags(a, t, now)})
	var lines []string
	for _, f := range fields {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("%-10s ", f.Name))+f.Value)
	}
	return lines
}

func (m model) screenLines(a agent.Agent) []string {
	if a.TmuxPane == "" {
		return []string{
			dimStyle.Render("This agent is not running in tmux, so its screen cannot be shown."),
			dimStyle.Render("Launch agents with n to see their screen here, attach and send prompts."),
		}
	}
	if m.screen.pane != a.TmuxPane {
		return []string{dimStyle.Render("capturing " + a.TmuxSession + " ...")}
	}
	if m.screen.err != nil {
		return []string{errStyle.Render(m.screen.err.Error())}
	}
	return strings.Split(strings.ReplaceAll(m.screen.text, "\t", "    "), "\n")
}

func processLines(a agent.Agent) []string {
	lines := []string{titleStyle.Render(fmt.Sprintf("%-7s %-7s %6s %7s  %s", "PID", "PPID", "CPU%", "MEM", "COMMAND"))}
	row := func(pid, ppid int, cpu float64, rss int64, depth int, cmd string) string {
		return fmt.Sprintf("%-7d %-7d %6.1f %7s  %s%s", pid, ppid, cpu, display.Bytes(rss), strings.Repeat("  ", depth), cmd)
	}
	lines = append(lines, row(a.PID, a.PPID, a.CPU, a.RSS, 0, a.Command))
	for _, c := range a.Children {
		lines = append(lines, row(c.PID, c.PPID, c.CPU, c.RSS, c.Depth, c.Command))
	}
	return append(lines, "", dimStyle.Render(fmt.Sprintf("total %.1f%% CPU, %s in %d processes",
		a.TreeCPU, display.Bytes(a.TreeRSS), len(a.Children)+1)))
}

func statsLines(a agent.Agent, t watch.Tracker, w, h int) []string {
	if len(t.History) < 2 {
		return []string{dimStyle.Render("collecting samples ...")}
	}
	cpu := make([]float64, len(t.History))
	mem := make([]float64, len(t.History))
	for i, s := range t.History {
		cpu[i], mem[i] = s.TreeCPU, float64(s.TreeRSS)
	}
	chartH := max((h-5)/2, 2)
	window := display.Duration(t.History[len(t.History)-1].Time.Sub(t.History[0].Time))
	now, avg, peak := stats(cpu)
	lines := []string{titleStyle.Render("CPU, agent and children") +
		dimStyle.Render(fmt.Sprintf("  now %.1f%%  avg %.1f%%  max %.1f%%  last %s", now, avg, peak, window))}
	lines = append(lines, barChart(cpu, w, chartH, scaleCPU(peak), func(v float64) string {
		return fmt.Sprintf("%.0f%%", v)
	})...)
	now, avg, peak = stats(mem)
	lines = append(lines, "", titleStyle.Render("Memory, agent and children")+
		dimStyle.Render(fmt.Sprintf("  now %s  avg %s  max %s", display.Bytes(int64(now)),
			display.Bytes(int64(avg)), display.Bytes(int64(peak)))))
	lines = append(lines, barChart(mem, w, chartH, scaleMem(peak), func(v float64) string {
		return display.Bytes(int64(v))
	})...)
	return lines
}

func (m model) footer() string {
	keys := footerList
	if m.focus == focusMain {
		keys = footerMain
	}
	right := dimStyle.Render("lazyai " + m.opts.Version)
	if m.msg != "" {
		right = m.msg
		if m.msgErr {
			right = errStyle.Render(m.msg)
		}
	}
	return spread(keyStyle.Render(keys), right, m.width)
}

func (m model) helpPopup() string {
	var lines []string
	for gi, g := range keyHelp {
		if gi > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, titleStyle.Render(g.title))
		for _, b := range g.bindings {
			lines = append(lines, keyStyle.Render(fmt.Sprintf("  %-20s", b.keys))+" "+b.desc)
		}
	}
	lines = append(lines, "", dimStyle.Render("press any key to close"))
	return popupBox("Keybindings", lines, min(76, m.width-4))
}

func (m model) menuPopup() string {
	var lines []string
	for i, it := range m.menu {
		row := fmt.Sprintf(" %-2s %s", it.key, it.label)
		if it.disabled != "" {
			row = dimStyle.Render(row + "  (" + it.disabled + ")")
		}
		if i == m.menuCursor {
			row = selStyle.Render(ansi.Strip(row))
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", dimStyle.Render("enter or key to run, esc to close"))
	title := "Menu"
	if a, ok := m.selected(); ok {
		title = fmt.Sprintf("%s %d", a.Kind, a.PID)
	}
	return popupBox(title, lines, min(90, m.width-4))
}

func (m model) confirmPopup() string {
	a := m.target
	verb := "Stop (SIGTERM)"
	if m.force {
		verb = "Kill (SIGKILL)"
	}
	lines := []string{
		fmt.Sprintf("%s %s %s %d", verb, a.Kind, a.Role, a.PID),
		"in " + display.Path(a.CWD) + "?",
		"",
		keyStyle.Render("y") + " yes    " + keyStyle.Render("any other key") + " no",
	}
	return popupBox("Confirm", lines, min(70, m.width-4))
}

func (m model) inputPopup() string {
	title, hint := "Launch", "KIND [DIR] [NAME] [-- ARGS]   kinds: "
	var kinds []string
	for _, k := range agent.LaunchKinds() {
		kinds = append(kinds, string(k))
	}
	hint += strings.Join(kinds, " ")
	if m.input == inputSend {
		title, hint = "Prompt for "+m.target.TmuxSession, "typed into the agent, then Enter"
	}
	w := min(80, m.width-4)
	text := string(m.buf) + "_"
	if tw := ansi.StringWidth(text); tw > w-4 {
		text = ansi.TruncateLeft(text, tw-(w-4), "")
	}
	lines := []string{text, "", dimStyle.Render(hint), dimStyle.Render("enter to submit, esc to cancel")}
	return popupBox(title, lines, w)
}

// box draws a bordered panel w by h with a title in the top border and an
// optional note in the bottom border. Content lines are cut or padded.
func box(title string, lines []string, w, h int, focused bool, note string) string {
	if w < 4 || h < 2 {
		return ""
	}
	b := lipgloss.RoundedBorder()
	color := dimColor
	if focused {
		color = focusColor
	}
	edge := lipgloss.NewStyle().Foreground(color)
	inner := w - 2
	t := ""
	if title != "" {
		t = ansi.Truncate(" "+title+" ", inner, "")
	}
	rows := []string{edge.Render(b.TopLeft) + t + edge.Render(strings.Repeat(b.Top, inner-ansi.StringWidth(t))) +
		edge.Render(b.TopRight)}
	for i := range h - 2 {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		rows = append(rows, edge.Render(b.Left)+fit(line, inner)+edge.Render(b.Right))
	}
	n := ""
	if note != "" {
		n = ansi.Truncate(" "+note+" ", inner, "")
	}
	rows = append(rows, edge.Render(b.BottomLeft)+edge.Render(strings.Repeat(b.Bottom, inner-ansi.StringWidth(n)))+
		dimStyle.Render(n)+edge.Render(b.BottomRight))
	return strings.Join(rows, "\n")
}

func popupBox(title string, lines []string, w int) string {
	w = max(w, 20)
	padded := make([]string, len(lines))
	for i, l := range lines {
		padded[i] = " " + l
	}
	return box(activeStyle.Render(title), padded, w, len(lines)+2, true, "")
}

// fit cuts or pads s to exactly w cells, closing any open style.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	return s + "\x1b[0m" + strings.Repeat(" ", max(w-ansi.StringWidth(s), 0))
}

// spread puts left and right at the two ends of a w-cell line; left is cut
// first when they do not fit.
func spread(left, right string, w int) string {
	rw := ansi.StringWidth(right)
	if rw >= w {
		return fit(right, w)
	}
	left = ansi.Truncate(left, max(w-rw-1, 0), "")
	gap := max(w-ansi.StringWidth(left)-rw, 1)
	return left + "\x1b[0m" + strings.Repeat(" ", gap) + right
}

// center overlays fg on the middle of bg.
func center(bg, fg string, w, h int) string {
	x := max((w-lipgloss.Width(fg))/2, 0)
	y := max((h-lipgloss.Height(fg))/2, 0)
	return overlay(bg, fg, x, y)
}

// overlay draws fg over bg with its top left corner at column x, row y.
func overlay(bg, fg string, x, y int) string {
	bl := strings.Split(bg, "\n")
	for i, line := range strings.Split(fg, "\n") {
		row := y + i
		if row < 0 || row >= len(bl) {
			continue
		}
		b := bl[row]
		left := ansi.Truncate(b, x, "")
		left += strings.Repeat(" ", max(x-ansi.StringWidth(left), 0))
		right := ansi.TruncateLeft(b, x+ansi.StringWidth(line), "")
		bl[row] = left + "\x1b[0m" + line + "\x1b[0m" + right
	}
	return strings.Join(bl, "\n")
}

func cropLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func baseDir(p string) string {
	if p == "" {
		return "-"
	}
	if short := display.Path(p); short == "~" {
		return short
	}
	return filepath.Base(p)
}
