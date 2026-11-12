package tui

import (
	"fmt"
	"regexp"
	"strconv"
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
	// A just-launched agent's row, for flashTime: yellow, and with a
	// yellow background while it is the selected row.
	flashStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	flashSelStyle = lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0")).Bold(true)
)

// minEvenListH is the smallest list panel (borders plus 3 rows) for which
// the column is split evenly.
const minEvenListH = 5

// Width of the compact list column in the detail layout: a fifth of the
// screen within these bounds, and never more than half of it.
const (
	navMinW = 24
	navMaxW = 32
)

// Screen layout: the top row takes screenTopPct percent of the body
// (screenTopDefault at start, + and - move it by screenTopStep between
// screenTopLow and screenTopHigh), at least screenTopMin rows, and the
// Screen panel below keeps at least screenMinH rows.
const (
	screenTopMin     = 8
	screenMinH       = 5
	screenLeftW      = 90 // widest list panel in the screen layout
	screenTopDefault = 20
	screenTopStep    = 10
	screenTopLow     = 10
	screenTopHigh    = 70
)

// layout is where each panel sits. Mouse handling uses it too.
type layout struct {
	leftW, rightW, bodyH int // rightW is 0 when the main panel is hidden
	mainH                int // height of the main panel: bodyH, or the top row
	statusH              int
	listY, listH         [numLists]int
	compact              bool // rows leave out CPU and memory, and status unless busy
	// The screen layout: one list panel with the lists as tabs in its
	// title, and a Screen panel screenH rows high at row screenY.
	tabbed           bool
	screenY, screenH int
}

func (m model) layout() layout {
	l := layout{bodyH: max(m.height-1, 6), statusH: 3}
	l.mainH = l.bodyH
	if m.mode == layoutScreen {
		return m.screenLayout(l)
	}
	switch m.mode {
	case layoutDetail:
		l.leftW = min(max(min(m.width/5, navMaxW), navMinW), m.width/2)
		l.rightW = m.width - l.leftW
		l.compact = true
	case layoutLists:
		l.leftW = m.width
	default:
		l.leftW = max(min(m.width/3, 56), min(32, m.width/2))
		l.rightW = max(m.width-l.leftW, 10)
	}
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

// screenLayout puts the status and the list in use (lists as tabs) left
// and the main panel right in a top row, and the Screen panel across the
// whole width below it.
func (m model) screenLayout(l layout) layout {
	l.tabbed = true
	l.leftW = max(min(m.width*3/5, screenLeftW), min(32, m.width/2))
	l.rightW = max(m.width-l.leftW, 10)
	top := max(l.bodyH*m.screenTopPct/100, screenTopMin)
	top = max(min(top, l.bodyH-screenMinH), min(screenMinH, l.bodyH))
	l.mainH = top
	l.screenY, l.screenH = top, l.bodyH-top
	l.statusH = 0 // the list panel's title and bottom border carry the status
	l.listY[m.lastList] = l.statusH
	l.listH[m.lastList] = top - l.statusH
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
	if l.statusH > 0 {
		left = append(left, m.statusPanel(l))
	}
	for i := range numLists {
		if l.listH[i] > 0 {
			left = append(left, m.listPanel(i, l))
		}
	}
	body := cropLines(strings.Join(left, "\n"), l.mainH)
	if l.rightW > 0 {
		body = cropLines(lipgloss.JoinHorizontal(lipgloss.Top, body, m.mainPanel(l)), l.mainH)
	}
	if l.screenH > 0 {
		body += "\n" + m.screenPanel(l)
	}
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
	case popupLaunchErr:
		body = center(body, m.launchErrPopup(), m.width, l.bodyH)
	}
	return body + "\n" + m.footer()
}

func (m model) statusPanel(l layout) string {
	line, right := m.statusText(l.leftW - 2)
	return box(titleStyle.Render("lazyai"), []string{spread(line, dimStyle.Render(right), l.leftW-2)},
		l.leftW, l.statusH, false, m.mode.String()+" layout")
}

// statusText is the status line in two parts, the counts and the memory
// and clock, in the longest form whose parts fit w cells with one space
// between them.
func (m model) statusText(w int) (string, string) {
	busy, inTmux, attached := 0, 0, 0
	var shown []agent.Agent
	for _, a := range m.all {
		if a.Role == agent.RoleHelper && !m.showHelpers {
			continue
		}
		shown = append(shown, a)
		if a.Status == "busy" {
			busy++
		}
		if a.TmuxPane != "" {
			inTmux++
			if a.TmuxClients > 0 {
				attached++
			}
		}
	}
	long := fmt.Sprintf("%d running, %d busy", len(shown), busy)
	short := fmt.Sprintf("%d run, %d busy", len(shown), busy)
	line, longAtt, shortAtt := long, long, short
	if inTmux > 0 {
		// Only with agents in tmux, where attached can be anything but 0.
		longAtt += fmt.Sprintf(", %d attached", attached)
		shortAtt += fmt.Sprintf(", %d att", attached)
		line = longAtt
	}
	// Shorter forms for a narrow column, so the counts are not cut.
	forms := [][2]string{{line, "loading ..."}, {short, "loading ..."}, {short, ""}}
	if !m.updated.IsZero() {
		mem, clock := display.Bytes(totalRSS(shown)), m.updated.Format("15:04:05")
		forms = [][2]string{
			{longAtt, "mem " + mem + " " + clock},
			{shortAtt, "mem " + mem + " " + clock},
			{longAtt, mem + " " + clock},
			{shortAtt, mem + " " + clock},
			{short, mem + " " + clock},
			{short, mem},
			{short, ""},
		}
	}
	right := ""
	for _, f := range forms {
		line, right = f[0], f[1]
		if len(line)+1+len(right) <= w {
			break
		}
	}
	return line, right
}

// totalRSS adds the tree memory of agents. An agent that runs below another
// one in the list is already in that one's tree and is not added again.
func totalRSS(agents []agent.Agent) int64 {
	scope := func(a agent.Agent) string {
		if a.ForeignPID {
			return a.ContainerID + "/"
		}
		return ""
	}
	below := map[string]bool{}
	for _, a := range agents {
		for _, c := range a.Children {
			below[scope(a)+strconv.Itoa(c.PID)] = true
		}
	}
	var sum int64
	for _, a := range agents {
		if !below[scope(a)+strconv.Itoa(a.PID)] {
			sum += a.TreeRSS
		}
	}
	return sum
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
		if m.filters[i] != "" {
			lines = append(lines, dimStyle.Render("no match"))
		} else {
			lines = append(lines, dimStyle.Render("none"))
		}
	}
	for r := top; r < len(lst.rows) && r < top+visible; r++ {
		a := lst.rows[r]
		t, _ := m.watcher.Tracker(a.Key())
		// The gutter marks agents in tmux: attach and prompt work on them.
		prefix := " " + string(a.Kind) + " "
		if disabledReason("a", a) == "" {
			mark := attachMark
			if a.TmuxClients > 0 {
				mark = attachedMark
			}
			prefix = mark + string(a.Kind) + " "
		}
		if i != listAgents {
			prefix += string(a.Role) + " "
		}
		suffix := ""
		if a.Protected {
			suffix = " *"
		}
		right := fmt.Sprintf("%s %5.1f%% %5s", a.Status, a.TreeCPU, display.Bytes(a.TreeRSS))
		if l.compact {
			// The main panel beside it has the numbers; the label gets the room.
			right = ""
			if a.Status == "busy" {
				right = a.Status
			}
		}
		room := inner - len(right) - 1 - ansi.StringWidth(prefix+suffix)
		lbl, isPath := rowLabel(a)
		if isPath {
			lbl = shortenPath(lbl, room)
		}
		flagged := false
		for _, f := range watch.Flags(a, t, now) {
			flagged = flagged || f != "self"
		}
		// The Claude name follows the tmux session only when it fits
		// whole; dim unless the whole row gets a style.
		if hint := nameHint(a); hint != "" && ansi.StringWidth(lbl)+1+ansi.StringWidth(hint) <= room {
			if r == lst.cursor || flagged || a.Status == "busy" {
				lbl += " " + hint
			} else {
				lbl += " " + dimStyle.Render(hint)
			}
		}
		row := spread(prefix+lbl+suffix, right, inner)
		switch {
		case m.flashing(a, now) && r == lst.cursor:
			row = flashSelStyle.Render(ansi.Strip(row))
		case m.flashing(a, now):
			row = flashStyle.Render(ansi.Strip(row))
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
	if l.tabbed {
		// One panel for the three lists: 1 2 3 pick the tab.
		var tabs []string
		for j := range numLists {
			t := fmt.Sprintf("%d %s", j+1, listTitles[j])
			if j == i {
				t = activeStyle.Render(t)
			}
			tabs = append(tabs, t)
		}
		title = titleStyle.Render("lazyai") + " - " + strings.Join(tabs, " - ")
		// The status line moves into the bottom border, before the position.
		room := inner - 2 - len(pos) - 3 - 2 // note padding, " - ", some border
		if line, right := m.statusText(room - 2); room > 10 {
			status := line
			if right != "" {
				status += " - " + right
			}
			if len(status) <= room {
				pos = strings.TrimSuffix(status+" - "+pos, " - ")
			}
		}
	}
	if f := m.filters[i]; f != "" || (m.filtering && i == m.lastList) {
		title += " " + keyStyle.Render("/"+f)
	}
	return box(title, lines, l.leftW, l.listH[i], focused, pos)
}

// mainSize is the number of content lines and how many fit.
func (m model) mainSize() (int, int) {
	l := m.layout()
	if l.rightW == 0 {
		return 0, 1
	}
	return len(m.mainLines(l.rightW - 2)), max(l.mainH-2, 1)
}

func (m model) mainPanel(l layout) string {
	inner := l.rightW - 2
	lines := m.mainLines(inner)
	visible := max(l.mainH-2, 1)
	maxScroll := max(len(lines)-visible, 0)
	off := min(m.scroll, maxScroll)
	if m.follow && m.tab == tabScreen {
		off = maxScroll
	}
	lines = lines[off:]

	var tabs []string
	for i, t := range tabTitles {
		if l.tabbed && i == tabScreen {
			continue // the Screen panel below shows it
		}
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
	return box(strings.Join(tabs, " - "), lines, l.rightW, l.mainH, m.focus == focusMain, pos)
}

// screenPanelLines is the Screen panel's content in the screen layout.
func (m model) screenPanelLines() []string {
	a, ok := m.selected()
	if !ok {
		return m.mainLines(0) // "no agents running" and the like
	}
	return m.screenLines(a)
}

// screenPanel is the Screen tab as a panel of its own across the whole
// width, below the lists and tabs of the screen layout. It follows the
// bottom like a log until scrolled.
func (m model) screenPanel(l layout) string {
	lines := m.screenPanelLines()
	visible := max(l.screenH-2, 1)
	maxScroll := max(len(lines)-visible, 0)
	off := min(m.screenScroll, maxScroll)
	if m.screenFollow {
		off = maxScroll
	}
	lines = lines[off:]
	title := "Screen"
	if a, ok := m.selected(); ok && a.TmuxSession != "" {
		// The Claude session name says what the pane is about; the box
		// cuts the title when the screen is too narrow for it.
		title += " " + dimStyle.Render(a.TmuxSession)
		if a.CWD != "" {
			title += " " + dimStyle.Render("- "+lastDirs(a.CWD))
		}
		if hint := nameHint(a); hint != "" {
			title += " " + dimStyle.Render("-") + " " + hint
		}
	}
	focused := m.focus == focusScreen
	if focused {
		title = activeStyle.Render("Screen") + strings.TrimPrefix(title, "Screen")
	}
	pos := ""
	if maxScroll > 0 {
		pos = fmt.Sprintf("%d-%d of %d", off+1, min(off+visible, off+len(lines)), off+len(lines))
	}
	return box(title, lines, m.width, l.screenH, focused, pos)
}

func (m model) mainLines(w int) []string {
	a, ok := m.selected()
	if !ok {
		if f := m.filters[m.lastList]; f != "" {
			return []string{dimStyle.Render("no " + strings.ToLower(listTitles[m.lastList]) + " match /" + f)}
		}
		return []string{dimStyle.Render("no " + strings.ToLower(listTitles[m.lastList]) + " running")}
	}
	switch m.tab {
	case tabScreen:
		return m.screenLines(a)
	case tabProcesses:
		return processLines(a)
	case tabStats:
		t, _ := m.watcher.Tracker(a.Key())
		return statsLines(a, t, w, max(m.layout().mainH-2, 4))
	}
	return m.infoLines(a)
}

func (m model) infoLines(a agent.Agent) []string {
	now := time.Now()
	t, _ := m.watcher.Tracker(a.Key())
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
	return screenText(m.screen.text)
}

// sgrRE matches one SGR escape sequence (colors and attributes).
var sgrRE = regexp.MustCompile(`^\x1b\[[0-9;:]*m`)

// screenText turns a colored capture into lines for a panel. Only SGR
// sequences are kept: any other escape sequence or control character
// from the pane is dropped, so pane content cannot move the cursor,
// retitle the terminal or write the clipboard. The colors in effect at
// the end of a line are repeated at the start of the next, since the
// panel resets them at every line end and a line may be shown without
// the ones above it.
func screenText(text string) []string {
	var lines []string
	var active []string // SGR sequences since the last reset
	for _, raw := range strings.Split(text, "\n") {
		var b strings.Builder
		b.WriteString(strings.Join(active, ""))
		for i := 0; i < len(raw); {
			c := raw[i]
			switch {
			case c == 0x1b:
				if m := sgrRE.FindString(raw[i:]); m != "" {
					b.WriteString(m)
					switch {
					case m == "\x1b[m" || m == "\x1b[0m":
						active = active[:0]
					case strings.HasPrefix(m, "\x1b[0;"):
						active = append(active[:0], m) // a reset, then new attributes
					default:
						if active = append(active, m); len(active) > 16 {
							active = active[len(active)-16:]
						}
					}
					i += len(m)
					continue
				}
				i += escLen(raw[i:])
				continue
			case c == '\t':
				b.WriteString("    ")
			case c < 0x20 || c == 0x7f:
				// other control characters are dropped
			default:
				b.WriteByte(c)
			}
			i++
		}
		lines = append(lines, b.String())
	}
	return lines
}

// escLen is the length of the escape sequence at the start of s (which
// begins with ESC), so it can be skipped: CSI up to its final byte, OSC
// and other strings up to BEL or ST, or ESC and one more byte.
func escLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}
	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
	case ']', 'P', '_', '^', 'X':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
	default:
		return 2
	}
	return len(s)
}

func processLines(a agent.Agent) []string {
	lines := []string{titleStyle.Render(fmt.Sprintf("%-7s %-7s %6s %7s  %s", "PID", "PPID", "CPU%", "MEM", "COMMAND"))}
	row := func(pid, ppid int, cpu float64, rss int64, depth int, cmd string) string {
		return fmt.Sprintf("%-7d %-7d %6.1f %7s  %s%s", pid, ppid, cpu, display.Bytes(rss), strings.Repeat("  ", depth), cmd)
	}
	// Parents first, dimmed: they show who started the agent and are not
	// in the totals. The agent is bold; its tree is indented below it.
	for i, p := range a.Parents {
		lines = append(lines, dimStyle.Render(row(p.PID, p.PPID, p.CPU, p.RSS, i, p.Command)))
	}
	top := len(a.Parents)
	lines = append(lines, titleStyle.Render(row(a.PID, a.PPID, a.CPU, a.RSS, top, a.Command)))
	for _, c := range a.Children {
		lines = append(lines, row(c.PID, c.PPID, c.CPU, c.RSS, top+c.Depth, c.Command))
	}
	total := fmt.Sprintf("total %.1f%% CPU, %s in %d processes", a.TreeCPU, display.Bytes(a.TreeRSS), len(a.Children)+1)
	if top > 0 {
		total += fmt.Sprintf("; the %d above it are not counted", top)
	}
	return append(lines, "", dimStyle.Render(total))
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
	switch {
	case m.filtering:
		keys = "filter /" + m.filters[m.lastList] + "_   enter keep  esc clear  up/down move"
	case m.focus == focusMain || m.focus == focusScreen:
		keys = footerMain
	case m.mode == layoutLists:
		keys = footerLists
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
			desc := b.desc
			if b.keys == layoutKey {
				desc += " (now " + m.mode.String() + ")"
			}
			lines = append(lines, keyStyle.Render(fmt.Sprintf("  %-20s", b.keys))+" "+desc)
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

// launchErrPopup shows the output of a launched command that exited at
// once, so the reason is readable after its tmux session is gone.
func (m model) launchErrPopup() string {
	e := m.launchErr
	w := min(100, m.width-4)
	lines := []string{errStyle.Render(fmt.Sprintf("%s exited right after start, status %d", e.Command, e.Status)), ""}
	if len(e.Output) == 0 {
		lines = append(lines, dimStyle.Render("(no output)"))
	}
	for _, l := range e.Output {
		lines = append(lines, ansi.Truncate(l, w-4, ".."))
	}
	lines = append(lines, "", dimStyle.Render("any key closes; up in n brings the line back"))
	return popupBox("Launch failed", lines, w)
}

func (m model) confirmPopup() string {
	a := m.target
	if m.confirmSend {
		return popupBox("Confirm", []string{
			fmt.Sprintf("%s is %s;", a.TmuxSession, display.Attached(a)),
			"someone may be typing there. Send a prompt anyway?",
			"",
			keyStyle.Render("y") + " yes    " + keyStyle.Render("any other key") + " no",
		}, min(70, m.width-4))
	}
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
	title, hint := "Launch", agent.LaunchUsage+"   kinds: "
	var kinds []string
	for _, k := range agent.LaunchKinds() {
		kinds = append(kinds, string(k))
	}
	hint += strings.Join(kinds, " ") + ", or a command on PATH"
	if m.input == inputSend {
		title, hint = "Prompt for "+m.target.TmuxSession, "typed into the agent, then Enter"
	}
	w := min(80, m.width-4)
	text := string(m.buf) + "_"
	if tw := ansi.StringWidth(text); tw > w-4 {
		text = ansi.TruncateLeft(text, tw-(w-4), "")
	}
	keys := "enter to submit, esc to cancel"
	if m.input == inputLaunch {
		keys = "up/down history, tab completes, " + keys
	}
	lines := []string{text, ""}
	if len(m.choices) > 0 {
		lines = append(lines, ansi.Truncate(strings.Join(m.choices, "  "), w-4, "..."), "")
	}
	lines = append(lines, dimStyle.Render(hint), dimStyle.Render(keys))
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

// lastDirs keeps the last two directories of p, so projects with the same
// directory name under different parents stay apart: "alpha/web-frontend".
func lastDirs(p string) string {
	if p == "" {
		return "-"
	}
	parts := strings.Split(display.Path(p), "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	if s := strings.Join(parts, "/"); s != "" {
		return s
	}
	return "/"
}

// shortenPath fits a lastDirs label into w cells by replacing the start
// of its last directory with "..", keeping the parent: "alpha/..nd". When
// even the parent does not fit, the start of the whole label goes.
func shortenPath(s string, w int) string {
	r := []rune(s)
	if len(r) <= w || w < 3 {
		return s
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		head := []rune(s[:i+1])
		if keep := w - len(head) - 2; keep >= 2 {
			return string(head) + ".." + string(r[len(r)-keep:])
		}
	}
	return ".." + string(r[len(r)-(w-2):])
}
