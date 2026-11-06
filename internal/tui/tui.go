// Package tui is the lazydocker-style dashboard: agent lists on the left,
// tabs about the selected agent on the right, popups for menus and prompts.
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazyai/internal/agent"
	"lazyai/internal/tmux"
	"lazyai/internal/watch"
)

// Options configures the dashboard.
type Options struct {
	Interval    time.Duration // refresh period
	LaunchDir   string        // default directory for new agents
	HistoryFile string        // launch history; "" keeps it in memory only
	// SettingsFile holds the layout saved with S; "" starts with the
	// defaults and S reports an error.
	SettingsFile string
	Version      string
}

// Run shows the dashboard until the user quits.
func Run(opts Options) error {
	_, err := tea.NewProgram(newModel(opts), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

type (
	agentsMsg struct {
		agents []agent.Agent
		err    error
	}
	screenMsg struct {
		pane, text string
		err        error
	}
	tickMsg   time.Time
	actionMsg struct {
		text string
		err  error
	}
)

// Left-hand list panels. focusMain is the main panel on the right.
const (
	listAgents = iota
	listServices
	listModels
	numLists
	focusMain   = numLists     // the main panel (tabs) on the right
	focusScreen = numLists + 1 // the Screen panel of the screen layout
)

var listTitles = [numLists]string{"Agents", "Services", "Models"}

// Main panel tabs.
const (
	tabInfo = iota
	tabScreen
	tabProcesses
	tabStats
	numTabs
)

var tabTitles = [numTabs]string{"Info", "Screen", "Processes", "Stats"}

// layoutMode is how the dashboard arranges the lists and the main panel;
// v cycles through the modes in this order (specs/tui.md, Layouts).
type layoutMode int

const (
	layoutSplit  layoutMode = iota // lists left, main panel right; the start layout
	layoutDetail                   // compact lists, wide main panel
	layoutLists                    // lists across the whole width, no main panel
	layoutScreen                   // tabbed list and tabs on top, Screen below
	numLayouts
)

var (
	layoutNames = [numLayouts]string{"split", "detail", "lists", "screen"}
	layoutHints = [numLayouts]string{
		"lists left, main panel right",
		"compact lists, wide main panel",
		"main panel hidden; v shows it",
		"lists and tabs on top, Screen below (+ / - resize)",
	}
)

func (l layoutMode) String() string { return layoutNames[l] }

// hiddenMainMsg answers keys for the main panel in the lists layout.
const hiddenMainMsg = "the lists layout hides the main panel; v shows it"

type popupKind int

const (
	popupNone popupKind = iota
	popupHelp
	popupMenu
	popupConfirm
	popupInput
)

type inputKind int

const (
	inputSend inputKind = iota
	inputLaunch
)

// screenHistory is how many lines above the visible pane Screen shows.
const screenHistory = 1000

// list is one left-hand panel. Selection follows the agent across refreshes.
type list struct {
	rows   []agent.Agent
	cursor int
	selKey string // agent.Key
}

func (l *list) set(rows []agent.Agent) {
	l.rows = rows
	l.cursor = min(l.cursor, len(rows)-1)
	for i, a := range rows {
		if a.Key() == l.selKey {
			l.cursor = i
		}
	}
	l.sync()
}

func (l *list) move(d int) {
	l.cursor += d
	l.sync()
}

func (l *list) sync() {
	l.cursor = max(min(l.cursor, len(l.rows)-1), 0)
	if len(l.rows) > 0 {
		l.selKey = l.rows[l.cursor].Key()
	}
}

type model struct {
	opts     Options
	all      []agent.Agent
	lists    [numLists]list
	focus    int // a list index or focusMain
	lastList int // list whose selection the main panel shows
	tab      int
	mode     layoutMode
	scroll   int  // main panel offset from the top
	follow   bool // Screen sticks to the bottom
	// screenScroll and screenFollow are scroll and follow of the Screen
	// panel of the screen layout, which shows beside the main panel.
	screenScroll int
	screenFollow bool
	// screenTopPct is the share of the body, in percent, that the top
	// row of the screen layout takes; the Screen panel gets the rest.
	screenTopPct int
	showHelpers  bool
	filters      [numLists]string // text a list's rows must match, "" for all
	filtering    bool             // keys go to the filter of lastList

	popup      popupKind
	menu       []menuItem
	menuCursor int
	target     agent.Agent // agent a confirm or send applies to
	force      bool
	// confirmSend makes the confirm popup ask before the prompt popup of
	// an attached agent, instead of before a stop.
	confirmSend bool
	input       inputKind
	buf         []rune
	hist        []string // launch lines, oldest first
	histPos     int      // index into hist; len(hist) is the line being typed
	draft       []rune   // the line being typed while browsing hist
	choices     []string // completions shown under the launch input

	msg     string
	msgErr  bool
	width   int
	height  int
	watcher *watch.Watcher
	loading bool
	updated time.Time

	screen    screenMsg // last capture of the selected agent's pane
	capturing bool
}

func newModel(opts Options) model {
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}
	m := model{opts: opts, watcher: watch.New(), loading: true, follow: true, screenFollow: true, screenTopPct: screenTopDefault, width: 120, height: 30,
		hist: loadHistory(opts.HistoryFile)}
	s, err := loadSettings(opts.SettingsFile)
	if err != nil {
		m.setMsg("settings ignored: "+err.Error(), true)
	}
	m.applySettings(s)
	return m
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
		cmds := []tea.Cmd{tick(m.opts.Interval), m.capture()}
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
		before, _ := m.selected()
		m.updated = time.Now()
		m.all = msg.agents
		m.watcher.Update(m.all, m.updated)
		m.regroup()
		if after, _ := m.selected(); after.Key() != before.Key() {
			return m, m.selectionChanged()
		}
	case screenMsg:
		m.capturing = false
		if a, ok := m.selected(); ok && a.TmuxPane == msg.pane && m.mode != layoutLists {
			m.screen = msg
		}
	case actionMsg:
		if msg.err != nil {
			m.setMsg(msg.err.Error(), true)
		} else {
			m.setMsg(msg.text, false)
		}
		return m, m.refresh()
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		switch m.popup {
		case popupHelp:
			m.popup = popupNone
			return m, nil
		case popupMenu:
			return m.updateMenu(msg)
		case popupConfirm:
			return m.updateConfirm(msg)
		case popupInput:
			return m.updateInput(msg)
		}
		return m.updateKey(msg)
	}
	return m, nil
}

// group says which list an agent belongs in.
func group(a agent.Agent) int {
	switch a.Role {
	case agent.RoleInteractive:
		return listAgents
	case agent.RoleEngine, agent.RoleJob:
		return listModels
	}
	return listServices
}

func (m *model) regroup() {
	var rows [numLists][]agent.Agent
	for _, a := range m.all {
		if a.Role == agent.RoleHelper && !m.showHelpers {
			continue
		}
		g := group(a)
		if !matchFilter(a, m.filters[g]) {
			continue
		}
		rows[g] = append(rows[g], a)
	}
	for i := range m.lists {
		m.lists[i].set(rows[i])
	}
}

func (m model) selected() (agent.Agent, bool) {
	l := m.lists[m.lastList]
	if len(l.rows) == 0 {
		return agent.Agent{}, false
	}
	return l.rows[l.cursor], true
}

// selectionChanged resets the main panel for a newly selected agent.
func (m *model) selectionChanged() tea.Cmd {
	m.scroll, m.follow = 0, true
	m.screenScroll, m.screenFollow = 0, true
	m.screen = screenMsg{}
	return m.capture()
}

// screenShown reports whether the selected agent's Screen is on the
// screen: the Screen tab, or the Screen panel of the screen layout.
func (m model) screenShown() bool {
	switch m.mode {
	case layoutLists:
		return false
	case layoutScreen:
		return true
	}
	return m.tab == tabScreen
}

// capture fetches the selected agent's tmux pane when Screen is shown.
func (m *model) capture() tea.Cmd {
	a, ok := m.selected()
	if !m.screenShown() || !ok || a.TmuxPane == "" || m.capturing {
		return nil
	}
	m.capturing = true
	pane := a.TmuxPane
	return func() tea.Msg {
		text, err := tmux.Capture(pane, screenHistory)
		return screenMsg{pane, text, err}
	}
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

func (m *model) focusList(i int) tea.Cmd {
	m.focus = i
	if i == m.lastList {
		return nil
	}
	m.lastList = i
	return m.selectionChanged()
}

func (m *model) cycleFocus(d int) tea.Cmd {
	if m.mode == layoutScreen {
		// One tabbed list, the main panel, the Screen panel.
		order := []int{m.lastList, focusMain, focusScreen}
		i := 0
		for j, f := range order {
			if f == m.focus {
				i = j
			}
		}
		m.focus = order[(i+d+len(order))%len(order)]
		return nil
	}
	n := numLists + 1
	if m.mode == layoutLists {
		n = numLists // the main panel is hidden
	}
	next := (m.focus + d + n) % n
	if next == focusMain {
		m.focus = focusMain
		return nil
	}
	return m.focusList(next)
}

// cycleLayout switches to the next layout. The lists layout hides the main
// panel, so it gives up the focus and its Screen capture, which would go
// stale unseen; leaving the lists layout captures again.
func (m *model) cycleLayout() tea.Cmd {
	m.mode = (m.mode + 1) % numLayouts
	m.setMsg(m.mode.String()+" layout: "+layoutHints[m.mode], false)
	if m.focus == focusScreen && m.mode != layoutScreen {
		m.focus = focusMain
	}
	if m.mode == layoutScreen && m.tab == tabScreen {
		// Screen has its own panel here; the tabs show the others.
		m.tab = tabInfo
		m.scroll = 0
	}
	if m.mode == layoutLists {
		if m.focus == focusMain {
			m.focus = m.lastList
		}
		m.screen = screenMsg{}
		return nil
	}
	return m.capture()
}

func (m *model) setTab(t int) tea.Cmd {
	d := 1
	if t < m.tab {
		d = -1
	}
	m.tab = (t + numTabs) % numTabs
	if m.mode == layoutScreen && m.tab == tabScreen {
		m.tab = (m.tab + d + numTabs) % numTabs // the Screen panel shows it
	}
	m.scroll, m.follow = 0, true
	return m.capture()
}

// scrollScreen moves the Screen panel of the screen layout by d lines;
// follow resumes at the bottom.
func (m *model) scrollScreen(d int) {
	total, visible := len(m.screenPanelLines()), max(m.layout().screenH-2, 1)
	maxScroll := max(total-visible, 0)
	if m.screenFollow {
		m.screenScroll = maxScroll
	}
	m.screenScroll = max(min(m.screenScroll+d, maxScroll), 0)
	m.screenFollow = m.screenScroll == maxScroll
}

// scrollFocused scrolls the focused panel, the main or the Screen panel.
func (m *model) scrollFocused(d int) {
	if m.focus == focusScreen {
		m.scrollScreen(d)
		return
	}
	m.scrollMain(d)
}

// scrollMain moves the main panel by d lines; follow resumes at the bottom.
func (m *model) scrollMain(d int) {
	total, visible := m.mainSize()
	maxScroll := max(total-visible, 0)
	if m.follow && m.tab == tabScreen {
		m.scroll = maxScroll
	}
	m.scroll = max(min(m.scroll+d, maxScroll), 0)
	m.follow = m.scroll == maxScroll
}

// resizeScreen moves the top row of the screen layout by d percent of the
// body, so the Screen panel below grows when d is negative.
func (m *model) resizeScreen(d int) {
	if m.mode != layoutScreen {
		m.setMsg("+ and - resize the Screen panel in the screen layout", false)
		return
	}
	m.screenTopPct = max(min(m.screenTopPct+d, screenTopHigh), screenTopLow)
	m.setMsg(fmt.Sprintf("Screen %d%%, top row %d%% (+ / - resize)", 100-m.screenTopPct, m.screenTopPct), false)
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	inMain := m.focus == focusMain || m.focus == focusScreen
	page := max(m.layout().mainH-4, 1)
	if m.focus == focusScreen {
		page = max(m.layout().screenH-4, 1)
	}
	if m.mode == layoutLists {
		switch key {
		case "enter", "]", "[", "pgdown", "pgup", "ctrl+d", "ctrl+u":
			m.setMsg(hiddenMainMsg, false)
			return m, nil
		}
	}
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.popup = popupHelp
	case "x":
		m.menu, m.menuCursor, m.popup = m.menuItems(), 0, popupMenu
	case "tab":
		return m, m.cycleFocus(1)
	case "shift+tab":
		return m, m.cycleFocus(-1)
	case "1", "2", "3":
		return m, m.focusList(int(key[0] - '1'))
	case "enter":
		m.focus = focusMain
	case "/":
		m.focus, m.filtering = m.lastList, true
	case "esc":
		if inMain {
			m.focus = m.lastList
		} else if m.filters[m.focus] != "" {
			return m, m.setFilter(m.focus, "")
		}
	case "+", "=":
		m.resizeScreen(-screenTopStep)
	case "-":
		m.resizeScreen(screenTopStep)
	case "]":
		return m, m.setTab(m.tab + 1)
	case "[":
		return m, m.setTab(m.tab - 1)
	case "j", "down", "k", "up":
		d := 1
		if key == "k" || key == "up" {
			d = -1
		}
		if inMain {
			m.scrollFocused(d)
			return m, nil
		}
		m.lists[m.focus].move(d)
		return m, m.selectionChanged()
	case "g", "home":
		if m.focus == focusScreen {
			m.screenScroll, m.screenFollow = 0, false
			return m, nil
		}
		if inMain {
			m.scroll, m.follow = 0, false
			return m, nil
		}
		m.lists[m.focus].move(-len(m.lists[m.focus].rows))
		return m, m.selectionChanged()
	case "G", "end":
		if inMain {
			m.scrollFocused(1 << 20)
			return m, nil
		}
		m.lists[m.focus].move(len(m.lists[m.focus].rows))
		return m, m.selectionChanged()
	case "pgdown", "ctrl+d":
		m.scrollFocused(page)
	case "pgup", "ctrl+u":
		m.scrollFocused(-page)
	default:
		return m.action(key)
	}
	return m, nil
}

// action runs one of the agent actions; keys and menu entries share it.
func (m model) action(key string) (tea.Model, tea.Cmd) {
	a, ok := m.selected()
	switch key {
	case "r":
		return m, m.refresh()
	case layoutKey:
		return m, m.cycleLayout()
	case saveKey:
		m.saveLayout()
	case "H":
		m.showHelpers = !m.showHelpers
		m.regroup()
		if m.showHelpers {
			m.setMsg("helpers shown in Services", false)
		} else {
			m.setMsg("helpers hidden", false)
		}
	case "n":
		m.popup, m.input, m.buf, m.draft, m.choices = popupInput, inputLaunch, nil, nil, nil
		m.histPos = len(m.hist)
	case "a", "p", "s", "K":
		if !ok {
			m.setMsg("nothing selected", true)
			break
		}
		if reason := disabledReason(key, a); reason != "" {
			m.setMsg(reason, true)
			break
		}
		switch key {
		case "a":
			session := a.TmuxSession
			return m, tea.ExecProcess(tmux.AttachCmd(session), func(err error) tea.Msg {
				return actionMsg{"back from " + session, err}
			})
		case "p":
			if a.TmuxClients > 0 {
				// Someone may be typing in the pane: the texts would mix.
				m.popup, m.target, m.confirmSend = popupConfirm, a, true
				break
			}
			m.popup, m.input, m.target, m.buf = popupInput, inputSend, a, nil
		default:
			m.popup, m.target, m.force, m.confirmSend = popupConfirm, a, key == "K", false
		}
	}
	return m, nil
}

// disabledReason explains why an action cannot apply to a, or "".
func disabledReason(key string, a agent.Agent) string {
	switch key {
	case "a", "p":
		if a.TmuxPane == "" {
			return "not running in tmux; start agents with n to attach or send prompts"
		}
	case "s", "K":
		if a.Protected {
			return agent.ErrProtected.Error()
		}
	}
	return ""
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key := msg.String(); key {
	case "esc", "q", "x":
		m.popup = popupNone
	case "j", "down":
		m.menuCursor = min(m.menuCursor+1, len(m.menu)-1)
	case "k", "up":
		m.menuCursor = max(m.menuCursor-1, 0)
	case "enter":
		m.popup = popupNone
		return m.action(m.menu[m.menuCursor].key)
	default:
		for _, it := range m.menu {
			if it.key == key {
				m.popup = popupNone
				return m.action(key)
			}
		}
	}
	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.popup = popupNone
	send := m.confirmSend
	m.confirmSend = false
	if msg.String() != "y" {
		m.setMsg("cancelled", false)
		return m, nil
	}
	if send {
		m.popup, m.input, m.buf = popupInput, inputSend, nil
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
	m.choices = nil
	if m.input == inputLaunch {
		switch msg.Type {
		case tea.KeyUp:
			m.historyMove(-1)
			return m, nil
		case tea.KeyDown:
			m.historyMove(1)
			return m, nil
		case tea.KeyTab:
			line, choices := completeLaunch(string(m.buf), m.opts.LaunchDir)
			m.buf, m.choices = []rune(line), choices
			return m, nil
		}
	}
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.popup = popupNone
		m.setMsg("cancelled", false)
		return m, nil
	case tea.KeyEnter:
		m.popup = popupNone
		return m, m.submit(strings.TrimSpace(string(m.buf)))
	case tea.KeyBackspace, tea.KeyCtrlH:
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
	if m.input == inputSend {
		a := m.target
		return func() tea.Msg {
			err := tmux.Send(a.TmuxPane, text)
			return actionMsg{fmt.Sprintf("sent %d chars to %s", len([]rune(text)), a.TmuxSession), err}
		}
	}
	// Every line goes into the history, also one that does not parse, so
	// a mistyped line can be fetched with up in the next popup and fixed.
	if m.opts.HistoryFile != "" {
		m.hist = loadHistory(m.opts.HistoryFile) // keep lines other lazyai instances added
	}
	m.hist = addHistory(m.hist, text)
	histErr := saveHistory(m.opts.HistoryFile, m.hist)
	spec, err := parseLaunch(text, m.opts.LaunchDir)
	if err != nil {
		m.setMsg(err.Error()+" (up in n brings the line back)", true)
		return nil
	}
	if histErr != nil {
		m.setMsg("launch history: "+histErr.Error(), true)
	}
	return func() tea.Msg {
		session, err := agent.Launch(spec.Kind, spec.Dir, spec.Name, spec.Args)
		return actionMsg{"launched " + session + " (select it and press a to attach)", err}
	}
}

func (m model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.popup != popupNone {
		return m, nil
	}
	l := m.layout()
	onScreen := l.screenH > 0 && msg.Y >= l.screenY
	onLeft := msg.X < l.leftW && !onScreen
	hit := -1
	for i := range numLists {
		if onLeft && msg.Y >= l.listY[i] && msg.Y < l.listY[i]+l.listH[i] {
			hit = i
		}
	}
	switch {
	case msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown:
		d := 1
		if msg.Button == tea.MouseButtonWheelUp {
			d = -1
		}
		if onScreen {
			m.scrollScreen(3 * d)
			return m, nil
		}
		if !onLeft {
			m.scrollMain(3 * d)
			return m, nil
		}
		if hit >= 0 {
			cmd := m.focusList(hit)
			m.lists[hit].move(d)
			return m, tea.Batch(cmd, m.selectionChanged())
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		if onScreen {
			m.focus = focusScreen
			return m, nil
		}
		if !onLeft {
			m.focus = focusMain
			return m, nil
		}
		if hit < 0 {
			return m, nil
		}
		cmd := m.focusList(hit)
		row := msg.Y - l.listY[hit] - 1
		if idx := m.listTop(hit, l) + row; row >= 0 && idx < len(m.lists[hit].rows) {
			m.lists[hit].cursor = idx
			m.lists[hit].sync()
			return m, tea.Batch(cmd, m.selectionChanged())
		}
		return m, cmd
	}
	return m, nil
}

// tmuxState is the word a filter finds an agent in tmux by: "attached" or
// "detached"; "" outside tmux.
func tmuxState(a agent.Agent) string {
	switch {
	case a.TmuxPane == "":
		return ""
	case a.TmuxClients > 0:
		return "attached"
	}
	return "detached"
}

// matchFilter reports whether a contains every word of filter, ignoring
// case, in its kind, role, label, directory, sessions, status, PID or
// tmux state (attached, detached).
func matchFilter(a agent.Agent, filter string) bool {
	words := strings.Fields(strings.ToLower(filter))
	if len(words) == 0 {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{string(a.Kind), string(a.Role), label(a), a.CWD,
		a.Name, a.TmuxSession, a.Container, a.Status, strconv.Itoa(a.PID), tmuxState(a)}, " "))
	for _, w := range words {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}

// setFilter changes the filter of list i and regroups the rows.
func (m *model) setFilter(i int, f string) tea.Cmd {
	before, _ := m.selected()
	m.filters[i] = f
	m.regroup()
	if after, _ := m.selected(); after.Key() != before.Key() {
		return m.selectionChanged()
	}
	return nil
}

// updateFilter edits the filter of the list in use while "/" is active.
// Enter keeps the filter, esc clears it; up and down move the selection.
func (m model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	i := m.lastList
	f := []rune(m.filters[i])
	switch msg.Type {
	case tea.KeyEnter:
		m.filtering = false
		return m, nil
	case tea.KeyEsc, tea.KeyCtrlC:
		m.filtering = false
		return m, m.setFilter(i, "")
	case tea.KeyUp, tea.KeyDown:
		d := 1
		if msg.Type == tea.KeyUp {
			d = -1
		}
		m.lists[i].move(d)
		return m, m.selectionChanged()
	case tea.KeyBackspace, tea.KeyCtrlH:
		if len(f) > 0 {
			f = f[:len(f)-1]
		}
	case tea.KeyCtrlU:
		f = nil
	case tea.KeySpace:
		f = append(f, ' ')
	case tea.KeyRunes:
		f = append(f, msg.Runes...)
	default:
		return m, nil
	}
	return m, m.setFilter(i, string(f))
}

// parseLaunch reads a launch line typed into the n popup.
func parseLaunch(line, defaultDir string) (agent.LaunchSpec, error) {
	return agent.ParseLaunch(strings.Fields(line), defaultDir)
}
