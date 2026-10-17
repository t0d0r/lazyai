// Package tui is the interactive agent dashboard.
package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"lazyai/internal/agent"
	"lazyai/internal/display"
	"lazyai/internal/tmux"
	"lazyai/internal/watch"
)

// Options configures the dashboard.
type Options struct {
	Interval  time.Duration // refresh period
	LaunchDir string        // default directory for new agents
}

// Run shows the dashboard until the user quits.
func Run(opts Options) error {
	_, err := tea.NewProgram(newModel(opts), tea.WithAltScreen()).Run()
	return err
}

type (
	agentsMsg struct {
		agents []agent.Agent
		err    error
	}
	tickMsg   time.Time
	actionMsg struct {
		text string
		err  error
	}
)

type mode int

const (
	modeList mode = iota
	modeConfirm
	modeInput
)

type inputKind int

const (
	inputSend inputKind = iota
	inputLaunch
)

type model struct {
	opts        Options
	all         []agent.Agent
	rows        []agent.Agent // all, minus helpers unless shown
	cursor      int
	selPID      int
	showHelpers bool
	detail      bool
	mode        mode
	target      agent.Agent // agent a confirm or send applies to
	force       bool
	input       inputKind
	buf         []rune
	msg         string
	msgErr      bool
	width       int
	height      int
	watcher     *watch.Watcher
	loading     bool
	updated     time.Time
}

func newModel(opts Options) model {
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}
	return model{opts: opts, watcher: watch.New(), loading: true, width: 120, height: 30}
}

func discover() tea.Msg {
	agents, err := agent.Discover()
	return agentsMsg{agents, err}
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Init() tea.Cmd {
	return tea.Batch(discover, tick(m.opts.Interval))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		cmds := []tea.Cmd{tick(m.opts.Interval)}
		if !m.loading {
			m.loading = true
			cmds = append(cmds, discover)
		}
		return m, tea.Batch(cmds...)
	case agentsMsg:
		m.loading = false
		if msg.err != nil {
			m.setMsg(msg.err.Error(), true)
			return m, nil
		}
		m.updated = time.Now()
		m.all = msg.agents
		m.watcher.Update(m.all, m.updated)
		m.applyFilter()
	case actionMsg:
		if msg.err != nil {
			m.setMsg(msg.err.Error(), true)
		} else {
			m.setMsg(msg.text, false)
		}
		return m, m.refresh()
	case tea.KeyMsg:
		switch m.mode {
		case modeConfirm:
			return m.updateConfirm(msg)
		case modeInput:
			return m.updateInput(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m *model) refresh() tea.Cmd {
	if m.loading {
		return nil
	}
	m.loading = true
	return discover
}

func (m *model) setMsg(s string, isErr bool) {
	m.msg, m.msgErr = s, isErr
}

func (m *model) applyFilter() {
	m.rows = nil
	for _, a := range m.all {
		if m.showHelpers || a.Role != agent.RoleHelper {
			m.rows = append(m.rows, a)
		}
	}
	m.cursor = 0
	for i, a := range m.rows {
		if a.PID == m.selPID {
			m.cursor = i
		}
	}
	m.syncSelection()
}

func (m *model) syncSelection() {
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(m.rows) > 0 {
		m.selPID = m.rows[m.cursor].PID
	}
}

func (m model) selected() (agent.Agent, bool) {
	if len(m.rows) == 0 {
		return agent.Agent{}, false
	}
	return m.rows[m.cursor], true
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.cursor--
		m.syncSelection()
	case "down", "j":
		m.cursor++
		m.syncSelection()
	case "home", "g":
		m.cursor = 0
		m.syncSelection()
	case "end", "G":
		m.cursor = len(m.rows) - 1
		m.syncSelection()
	case "enter", "d":
		m.detail = !m.detail
	case "h":
		m.showHelpers = !m.showHelpers
		m.applyFilter()
	case "r":
		return m, m.refresh()
	case "a":
		a, ok := m.selected()
		if !ok {
			break
		}
		if a.TmuxSession == "" {
			m.setMsg("not running in tmux; start agents with n to attach to them", true)
			break
		}
		session := a.TmuxSession
		return m, tea.ExecProcess(tmux.AttachCmd(session), func(err error) tea.Msg {
			return actionMsg{"back from " + session, err}
		})
	case "p":
		a, ok := m.selected()
		if !ok {
			break
		}
		if a.TmuxPane == "" {
			m.setMsg("not running in tmux; only agents in tmux can receive prompts", true)
			break
		}
		m.mode, m.input, m.target, m.buf = modeInput, inputSend, a, nil
	case "n":
		m.mode, m.input, m.buf = modeInput, inputLaunch, nil
	case "s", "K":
		a, ok := m.selected()
		if !ok {
			break
		}
		if a.Protected {
			m.setMsg(agent.ErrProtected.Error(), true)
			break
		}
		m.mode, m.target, m.force = modeConfirm, a, msg.String() == "K"
	}
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList
	if msg.String() != "y" {
		m.setMsg("cancelled", false)
		return m, nil
	}
	a, force := m.target, m.force
	verb := "stopped"
	if force {
		verb = "killed"
	}
	m.setMsg(fmt.Sprintf("stopping %s %d ...", a.Kind, a.PID), false)
	return m, func() tea.Msg {
		err := agent.Stop(a, force, agent.StopTimeout)
		return actionMsg{fmt.Sprintf("%s %s %d", verb, a.Kind, a.PID), err}
	}
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.mode = modeList
		m.setMsg("cancelled", false)
		return m, nil
	case tea.KeyEnter:
		m.mode = modeList
		return m, m.submit(strings.TrimSpace(string(m.buf)))
	case tea.KeyBackspace:
		if len(m.buf) > 0 {
			m.buf = m.buf[:len(m.buf)-1]
		}
	case tea.KeyCtrlU:
		m.buf = nil
	case tea.KeySpace:
		m.buf = append(m.buf, ' ')
	case tea.KeyRunes:
		m.buf = append(m.buf, msg.Runes...)
	}
	return m, nil
}

func (m *model) submit(text string) tea.Cmd {
	if text == "" {
		m.setMsg("cancelled", false)
		return nil
	}
	switch m.input {
	case inputSend:
		a := m.target
		return func() tea.Msg {
			err := tmux.Send(a.TmuxPane, text)
			return actionMsg{fmt.Sprintf("sent %d chars to %s", len([]rune(text)), a.TmuxSession), err}
		}
	default:
		kind, dir, name, extra, err := parseLaunch(text, m.opts.LaunchDir)
		if err != nil {
			m.setMsg(err.Error(), true)
			return nil
		}
		return func() tea.Msg {
			session, err := agent.Launch(kind, dir, name, extra)
			return actionMsg{"launched " + session + " (press a on it to attach)", err}
		}
	}
}

// parseLaunch reads "KIND [DIR] [NAME] [-- ARGS...]".
func parseLaunch(line, defaultDir string) (agent.Kind, string, string, []string, error) {
	head, tail, _ := strings.Cut(" "+line+" ", " -- ")
	words := strings.Fields(head)
	if len(words) == 0 || len(words) > 3 {
		return "", "", "", nil, errors.New("usage: KIND [DIR] [NAME] [-- ARGS...]")
	}
	kind := agent.Kind(words[0])
	if _, err := agent.LaunchCommand(kind, []string{"check"}); err != nil {
		return "", "", "", nil, err
	}
	dir := defaultDir
	if len(words) > 1 {
		dir = expandDir(words[1], defaultDir)
	}
	name := ""
	if len(words) > 2 {
		name = words[2]
	}
	return kind, dir, name, strings.Fields(tail), nil
}

func expandDir(p, base string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(base, p)
	}
	return filepath.Clean(p)
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	headStyle  = lipgloss.NewStyle().Bold(true).Underline(true)
	selStyle   = lipgloss.NewStyle().Reverse(true)
	busyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	flagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle   = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
)

type column struct {
	title string
	width int
}

var columns = []column{
	{"PID", 7}, {"KIND", 12}, {"ROLE", 11}, {"STATUS", 11}, {"CPU%", 6},
	{"MEM", 7}, {"UP", 7}, {"TMUX", 20}, {"FLAGS", 16}, {"WHERE", 0},
}

func (m model) View() string {
	var b strings.Builder
	now := time.Now()

	busy := 0
	for _, a := range m.rows {
		if a.Status == "busy" {
			busy++
		}
	}
	helpers := "helpers hidden"
	if m.showHelpers {
		helpers = "helpers shown"
	}
	state := "updated " + m.updated.Format("15:04:05")
	if m.updated.IsZero() {
		state = "loading ..."
	}
	b.WriteString(titleStyle.Render("lazyai") +
		dimStyle.Render(fmt.Sprintf("  %d agents, %d busy  |  %s  |  %s", len(m.rows), busy, helpers, state)) + "\n\n")

	cells := make([]string, len(columns))
	for i, c := range columns {
		cells[i] = c.title
	}
	b.WriteString(headStyle.Render(m.formatRow(cells)) + "\n")

	detailLines := 0
	if m.detail {
		detailLines = 10
	}
	visible := max(m.height-7-detailLines, 3)
	top := max(m.cursor-visible+1, 0)
	if len(m.rows) == 0 && !m.updated.IsZero() {
		b.WriteString(dimStyle.Render("no agents running") + "\n")
	}
	for i := top; i < len(m.rows) && i < top+visible; i++ {
		a := m.rows[i]
		t, _ := m.watcher.Tracker(a.PID)
		tmuxCol := strings.TrimPrefix(a.TmuxSession, tmux.Prefix)
		if tmuxCol == "" {
			tmuxCol = "-"
		}
		fl := strings.Join(watch.Flags(a, t, now), ",")
		row := m.formatRow([]string{
			fmt.Sprint(a.PID), string(a.Kind), string(a.Role), display.Status(a, t, now),
			fmt.Sprintf("%.1f", a.CPU), display.Bytes(a.RSS), display.Duration(now.Sub(a.Start)),
			tmuxCol, fl, display.Where(a),
		})
		switch {
		case i == m.cursor:
			row = selStyle.Render(row)
		case fl != "" && fl != "self":
			row = flagStyle.Render(row)
		case a.Status == "busy":
			row = busyStyle.Render(row)
		}
		b.WriteString(row + "\n")
	}

	if m.detail {
		if a, ok := m.selected(); ok {
			b.WriteString("\n" + m.detailView(a))
		}
	}

	b.WriteString("\n")
	if m.msg != "" {
		if m.msgErr {
			b.WriteString(errStyle.Render(m.msg))
		} else {
			b.WriteString(m.msg)
		}
	}
	b.WriteString("\n")
	b.WriteString(m.footer())
	return b.String()
}

func (m model) formatRow(cells []string) string {
	fixed := 0
	for _, c := range columns {
		fixed += c.width + 1
	}
	var b strings.Builder
	for i, c := range columns {
		w := c.width
		if w == 0 {
			w = max(m.width-fixed, 10)
		}
		cell := display.Truncate(cells[i], w)
		b.WriteString(cell)
		if i < len(columns)-1 {
			b.WriteString(strings.Repeat(" ", w-len([]rune(cell))+1))
		}
	}
	return b.String()
}

func (m model) detailView(a agent.Agent) string {
	w := max(m.width-12, 20)
	var b strings.Builder
	for _, f := range display.Details(a) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-10s ", f.Name)) + display.Truncate(f.Value, w) + "\n")
	}
	return b.String()
}

func (m model) footer() string {
	switch m.mode {
	case modeConfirm:
		a := m.target
		verb := "Stop (SIGTERM)"
		if m.force {
			verb = "Force kill (SIGKILL)"
		}
		return errStyle.Render(fmt.Sprintf("%s %s %d in %s? y/N", verb, a.Kind, a.PID, display.Path(a.CWD)))
	case modeInput:
		prompt := "launch (KIND [DIR] [NAME] [-- ARGS]): "
		if m.input == inputSend {
			prompt = "prompt for " + m.target.TmuxSession + ": "
		}
		return titleStyle.Render(prompt) + string(m.buf) + "_" + dimStyle.Render("  (enter send, esc cancel)")
	}
	return dimStyle.Render("j/k move  enter details  a attach  p prompt  n new  s stop  K kill  h helpers  r refresh  q quit")
}
