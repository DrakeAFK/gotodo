package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	subTitleStyle = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#A8B2C8"))

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#bd93f9")).
		MarginTop(1).
		MarginBottom(0)

	headingStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFB86C")).
		MarginTop(1)

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F8F8F2")).
		Background(lipgloss.Color("#bd93f9")).
		PaddingLeft(1).
		PaddingRight(1)

	normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		PaddingLeft(1).
		PaddingRight(1)

	doneStyle = lipgloss.NewStyle().
		Faint(true).
		Strikethrough(true).
		Foreground(lipgloss.Color("#A8B2C8")).
		PaddingLeft(1).
		PaddingRight(1)

	checkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	progressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)

	treeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8B2C8"))
)

type mode int

const (
	modeList mode = iota
	modeAdd
	modeAddSubtask
	modeAddHeading
	modeHelp
	modeCommand
)

type Model struct {
	store Store
	state diskState

	cursor   int
	showDone bool
	mode     mode

	input    textinput.Model // add-task input
	cmdInput textinput.Model // ":" command bar input

	err error

	// for centering help modal
	width  int
	height int

	// tiny status line ("saved", "file set", etc)
	status    string
	statusSet time.Time
}

func NewModel(store Store, st diskState) Model {
	ti := textinput.New()
	ti.Placeholder = "task_007"
	ti.CharLimit = 200
	ti.Width = 50

	ci := textinput.New()
	ci.Prompt = ":"
	ci.Placeholder = "help | w | q | set file=~/... | toggle done"
	ci.CharLimit = 200
	ci.Width = 60

	return Model{
		store:    store,
		state:    st,
		cursor:   0,
		showDone: false,
		mode:     modeList,
		input:    ti,
		cmdInput: ci,
	}
}

type savedMsg struct{}
type saveErrMsg struct{ err error }
type loadedMsg struct{ st diskState }
type loadErrMsg struct{ err error }

func saveCmd(store Store, st diskState) tea.Cmd {
	return func() tea.Msg {
		if err := store.Save(st); err != nil {
			return saveErrMsg{err: err}
		}
		return savedMsg{}
	}
}

func loadCmd(store Store) tea.Cmd {
	return func() tea.Msg {
		st, err := store.Load()
		if err != nil {
			return loadErrMsg{err: err}
		}
		return loadedMsg{st: st}
	}
}

func (m Model) Init() tea.Cmd { return nil }

// --------------------
// helpers
// --------------------

func (m Model) visibleTasks() []Task {
	var filtered []Task
	for _, t := range m.state.Tasks {
		if m.showDone || !t.Done {
			filtered = append(filtered, t)
		}
	}

	hasID := make(map[int]bool)
	for _, t := range filtered {
		hasID[t.ID] = true
	}

	children := make(map[int][]Task)
	var root []Task

	for _, t := range filtered {
		if t.ParentID == 0 || !hasID[t.ParentID] {
			root = append(root, t)
		} else {
			children[t.ParentID] = append(children[t.ParentID], t)
		}
	}

	var out []Task
	var add func(t Task)
	add = func(t Task) {
		out = append(out, t)
		for _, child := range children[t.ID] {
			add(child)
		}
	}

	// Group root components by their project
	groupedProjects := make(map[string][]Task)
	var projectOrder []string // ensure stable deterministic ordering based on appearance
	
	for _, r := range root {
		p := r.Project
		if len(groupedProjects[p]) == 0 {
			projectOrder = append(projectOrder, p)
		}
		groupedProjects[p] = append(groupedProjects[p], r)
	}

	// Output items sorted by project groupings instead of purely chronological
	for _, p := range projectOrder {
		for _, r := range groupedProjects[p] {
			add(r)
		}
	}

	return out
}

func (m *Model) clampCursor() {
	n := len(m.visibleTasks())
	if n == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m *Model) setStatus(s string) {
	m.status = s
	m.statusSet = time.Now()
}

func (m *Model) addTask(title string, parentID int) tea.Cmd {
	title = strings.TrimSpace(title)
	if title == "" {
		m.setStatus("title is empty")
		return nil
	}

	var project string
	words := strings.Fields(title)
	var cleaned []string
	for _, w := range words {
		if strings.HasPrefix(w, "+") && len(w) > 1 {
			project = strings.TrimPrefix(w, "+")
		} else {
			cleaned = append(cleaned, w)
		}
	}
	title = strings.Join(cleaned, " ")
	if title == "" {
		title = project // fallback
	}

	t := Task{
		ID:        m.state.NextID,
		ParentID:  parentID,
		Project:   project,
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	}

	m.state.NextID++
	m.state.Tasks = append(m.state.Tasks, t)
	m.setStatus("adding…")
	return saveCmd(m.store, m.state)
}

func (m *Model) addHeading(title string) tea.Cmd {
	title = strings.TrimSpace(title)
	if title == "" {
		m.setStatus("heading name is empty")
		return nil
	}

	t := Task{
		ID:        m.state.NextID,
		Title:     title,
		IsHeading: true,
		CreatedAt: time.Now(),
	}

	m.state.NextID++
	m.state.Tasks = append(m.state.Tasks, t)
	m.setStatus("heading added")
	return saveCmd(m.store, m.state)
}

func (m *Model) toggleTaskDoneByVisibleIndex(i int) tea.Cmd {
	vis := m.visibleTasks()
	if i < 0 || i >= len(vis) {
		return nil
	}
	id := vis[i].ID

	for k := range m.state.Tasks {
		if m.state.Tasks[k].ID == id {
			m.state.Tasks[k].Done = !m.state.Tasks[k].Done
			if m.state.Tasks[k].Done {
				m.state.Tasks[k].DoneAt = time.Now()
			} else {
				m.state.Tasks[k].DoneAt = time.Time{}
			}
			break
		}
	}
	m.setStatus("saving…")
	return saveCmd(m.store, m.state)
}

func (m *Model) deleteTaskByVisibleIndex(i int) tea.Cmd {
	vis := m.visibleTasks()
	if i < 0 || i >= len(vis) {
		return nil
	}
	id := vis[i].ID

	next := make([]Task, 0, len(m.state.Tasks)-1)
	for _, t := range m.state.Tasks {
		if t.ID != id {
			next = append(next, t)
		}
	}
	m.state.Tasks = next
	m.setStatus("saving…")
	return saveCmd(m.store, m.state)
}

func (m *Model) swapTaskInState(idA, idB int) tea.Cmd {
	idxA, idxB := -1, -1
	for i, t := range m.state.Tasks {
		if t.ID == idA { idxA = i }
		if t.ID == idB { idxB = i }
	}
	if idxA >= 0 && idxB >= 0 {
		m.state.Tasks[idxA], m.state.Tasks[idxB] = m.state.Tasks[idxB], m.state.Tasks[idxA]
		m.setStatus("moved")
		return saveCmd(m.store, m.state)
	}
	return nil
}

func (m *Model) moveTaskUp(i int) tea.Cmd {
	vis := m.visibleTasks()
	if i <= 0 || i >= len(vis) { return nil }
	task := vis[i]
	
	// find previous sibling in visible list
	var prevSiblingTask Task
	found := false
	for j := i - 1; j >= 0; j-- {
		if vis[j].ParentID == task.ParentID {
			prevSiblingTask = vis[j]
			found = true
			break
		}
	}
	
	if !found { return nil }
	
	cmd := m.swapTaskInState(task.ID, prevSiblingTask.ID)
	// keep cursor on the same task logically, which now moved up
	// To do this simply, we decrement the cursor, but only by the number of elements it jumped
	// the simplest way is just m.cursor-- iteratively because it moves up past one sibling branch
	jumpCount := 1
	for j := i - 1; j >= 0; j-- {
		if vis[j].ID == prevSiblingTask.ID { break }
		jumpCount++
	}
	m.cursor -= jumpCount
	m.clampCursor()
	
	return cmd
}

func (m *Model) moveTaskDown(i int) tea.Cmd {
	vis := m.visibleTasks()
	if i < 0 || i >= len(vis)-1 {
		return nil
	}
	task := vis[i]

	var nextSiblingTask Task
	found := false
	for j := i + 1; j < len(vis); j++ {
		if vis[j].ParentID == task.ParentID {
			nextSiblingTask = vis[j]
			found = true
			break
		}
	}

	if !found {
		return nil
	}

	cmd := m.swapTaskInState(task.ID, nextSiblingTask.ID)

	// Recompute visible list after swap, find where our task landed
	newVis := m.visibleTasks()
	for idx, v := range newVis {
		if v.ID == task.ID {
			m.cursor = idx
			break
		}
	}
	m.clampCursor()
	return cmd
}

func (m *Model) depth(id int) int {
	d := 0
	curr := id
	for curr != 0 {
		found := false
		for _, t := range m.state.Tasks {
			if t.ID == curr {
				if t.ParentID == curr { break }
				curr = t.ParentID
				found = true
				if curr != 0 { d++ }
				break
			}
		}
		if !found { break }
	}
	return d
}

func (m *Model) promoteTask(i int) tea.Cmd {
	vis := m.visibleTasks()
	if i < 0 || i >= len(vis) { return nil }
	task := vis[i]
	if task.ParentID == 0 { return nil }
	
	for k := range m.state.Tasks {
		if m.state.Tasks[k].ID == task.ID {
			var pParent int
			for _, pt := range m.state.Tasks {
				if pt.ID == task.ParentID { pParent = pt.ParentID; break }
			}
			m.state.Tasks[k].ParentID = pParent
			break
		}
	}
	m.setStatus("promoted")
	return saveCmd(m.store, m.state)
}

func (m *Model) demoteTask(i int) tea.Cmd {
	vis := m.visibleTasks()
	if i <= 0 || i >= len(vis) { return nil }
	
	task := vis[i]
	prev := vis[i-1]
	
	curr := prev.ID
	for curr != 0 {
		if curr == task.ID { return nil }
		next := 0
		for _, t := range m.state.Tasks {
			if t.ID == curr { next = t.ParentID; break }
		}
		if next == curr { break }
		curr = next
	}

	for k := range m.state.Tasks {
		if m.state.Tasks[k].ID == task.ID {
			m.state.Tasks[k].ParentID = prev.ID
			break
		}
	}
	m.setStatus("demoted")
	return saveCmd(m.store, m.state)
}


// --------------------
// bubbletea
// --------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case saveErrMsg:
		m.err = msg.err
		m.setStatus("save failed")
		return m, nil
	case savedMsg:
		m.err = nil
		m.setStatus("saved")
		return m, nil

	case loadErrMsg:
		m.err = msg.err
		m.setStatus("load failed")
		return m, nil
	case loadedMsg:
		m.err = nil
		m.state = msg.st
		m.clampCursor()
		m.setStatus("reloaded")
		return m, nil
	}

	switch m.mode {
	case modeAdd, modeAddSubtask, modeAddHeading:
		return m.updateAdd(msg)
	case modeHelp:
		return m.updateHelp(msg)
	case modeCommand:
		return m.updateCommand(msg)
	default:
		return m.updateList(msg)
	}
}

func (m Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "?":
			m.mode = modeList
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.mode = modeList
			m.input.Blur()
			m.input.SetValue("")
			m.setStatus("canceled")
			return m, nil
		case "enter":
			if m.mode == modeAddHeading {
				cmd = m.addHeading(m.input.Value())
				m.mode = modeList
				m.input.Blur()
				m.input.SetValue("")
				return m, cmd
			}
			parentID := 0
			if m.mode == modeAddSubtask {
				vis := m.visibleTasks()
				if m.cursor >= 0 && m.cursor < len(vis) {
					parentID = vis[m.cursor].ID
				}
			}
			cmd = m.addTask(m.input.Value(), parentID)
			m.mode = modeList
			m.input.Blur()
			m.input.SetValue("")
			return m, cmd
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateCommand(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.mode = modeList
			m.cmdInput.Blur()
			m.cmdInput.SetValue("")
			m.setStatus("canceled")
			return m, nil
		case "enter":
			line := strings.TrimSpace(m.cmdInput.Value())
			m.cmdInput.Blur()
			m.cmdInput.SetValue("")
			m.mode = modeList
			return m, m.execCommand(line)
		}
	}

	m.cmdInput, cmd = m.cmdInput.Update(msg)
	return m, cmd
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "?":
			m.mode = modeHelp
			return m, nil

		case ":":
			m.mode = modeCommand
			m.cmdInput.Focus()
			m.cmdInput.SetValue("")
			return m, nil

		case "j", "down":
			m.cursor++
			m.clampCursor()
			return m, nil
		case "k", "up":
			m.cursor--
			m.clampCursor()
			return m, nil

		case "a":
			m.mode = modeAdd
			m.input.Placeholder = "task_007"
			m.input.Focus()
			return m, nil

		case "s":
			vis := m.visibleTasks()
			if len(vis) > 0 {
				m.mode = modeAddSubtask
				m.input.Placeholder = "subtask name"
				m.input.Focus()
				return m, nil
			}

		case " ":
			vis := m.visibleTasks()
			if m.cursor >= 0 && m.cursor < len(vis) && vis[m.cursor].IsHeading {
				return m, nil // headings cannot be toggled done
			}
			cmd := m.toggleTaskDoneByVisibleIndex(m.cursor)
			m.clampCursor()
			return m, cmd

		case "d", "backspace":
			cmd := m.deleteTaskByVisibleIndex(m.cursor)
			m.clampCursor()
			return m, cmd

		case "g":
			m.mode = modeAddHeading
			m.input.Placeholder = "heading name"
			m.input.Focus()
			return m, nil

		case "t":
			m.showDone = !m.showDone
			m.clampCursor()
			m.setStatus("toggled view")
			return m, nil

		case "L":
			cmd := m.demoteTask(m.cursor)
			return m, cmd
		case "H":
			cmd := m.promoteTask(m.cursor)
			return m, cmd
		case "K":
			return m, m.moveTaskUp(m.cursor)
		case "J":
			return m, m.moveTaskDown(m.cursor)
		}
	}

	return m, nil
}

// --------------------
// ":" commands
// --------------------

func (m *Model) execCommand(line string) tea.Cmd {
	if line == "" {
		return nil
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "q", "quit", "exit":
		return tea.Quit

	case "w", "write", "save":
		m.setStatus("saving…")
		return saveCmd(m.store, m.state)

	case "help", "h":
		m.mode = modeHelp
		return nil

	case "toggle":
		if len(parts) >= 2 && parts[1] == "done" {
			m.showDone = !m.showDone
			m.clampCursor()
			m.setStatus("toggled done view")
			return nil
		}

	case "set":
		// set file=~/...   OR   set file ~/...
		val := ""
		if len(parts) >= 2 && strings.HasPrefix(parts[1], "file=") {
			val = strings.TrimPrefix(parts[1], "file=")
		} else if len(parts) >= 3 && parts[1] == "file" {
			val = parts[2]
		}

		val = strings.TrimSpace(val)
		if val == "" {
			m.setStatus("usage: set file=~/path/to/tasks.json")
			return nil
		}

		p, err := expandHome(val)
		if err != nil {
			m.err = err
			m.setStatus("failed to expand path")
			return nil
		}

		m.store.Path = p
		m.setStatus("file set; reloading…")
		return loadCmd(m.store)
	}

	m.setStatus("unknown command: " + line)
	return nil
}

// --------------------
// view
// --------------------

func (m Model) View() string {
	if m.mode == modeHelp {
		return m.helpModalView()
	}

	title := titleStyle.Render("gotodo")
	sub := subTitleStyle.Render("keyboard-driven task manager")

	// Task counter
	var total, done int
	for _, t := range m.state.Tasks {
		if !t.IsHeading {
			total++
			if t.Done {
				done++
			}
		}
	}
	counterText := fmt.Sprintf("%d/%d done", done, total)
	if total > 0 {
		pct := float64(done) / float64(total)
		barLen := 10
		filled := int(pct * float64(barLen))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)
		counterText = progressStyle.Render(bar) + " " + subTitleStyle.Render(counterText)
	} else {
		counterText = subTitleStyle.Render(counterText)
	}

	header := fmt.Sprintf("%s  %s  %s\n\n", title, sub, counterText)

	// Add / heading mode
	if m.mode == modeAdd || m.mode == modeAddSubtask || m.mode == modeAddHeading {
		help := subTitleStyle.Render("enter=save  esc=cancel")
		modeName := "Add task"
		switch m.mode {
		case modeAddSubtask:
			modeName = "Add subtask"
		case modeAddHeading:
			modeName = "Add heading"
		}
		base := header +
			lipgloss.NewStyle().Bold(true).Render(modeName) + "\n" +
			m.input.View() + "\n" +
			help + "\n\n" +
			m.footer()
		return base
	}

	// List body
	vis := m.visibleTasks()

	var body string
	if len(vis) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true).
			Render("nothing here — press  a  to add a task,  g  for a heading")
		body = "\n" + emptyMsg + "\n\n"
	} else {
		var b strings.Builder
		var lastProject string

		for i, t := range vis {
			// Heading rows render as styled separators
			if t.IsHeading {
				selected := i == m.cursor
				headingText := fmt.Sprintf("── %s ", strings.ToUpper(t.Title))
				pad := 40 - len(headingText)
				if pad > 0 {
					headingText += strings.Repeat("─", pad)
				}
				if selected {
					line := lipgloss.NewStyle().
						Foreground(lipgloss.Color("#bd93f9")).
						Bold(true).
						Render("▶ ") + selectedStyle.Render(headingText)
					b.WriteString(line + "\n")
				} else {
					b.WriteString("  " + headingStyle.Render(headingText) + "\n")
				}
				continue
			}

			// Print project header before the first root item of a new project group
			if t.ParentID == 0 && !t.IsHeading {
				if t.Project != lastProject {
					lastProject = t.Project
					if lastProject != "" {
						headerText := fmt.Sprintf("─── %s ", strings.ToUpper(lastProject))
						pad := 40 - len(headerText)
						if pad > 0 {
							headerText += strings.Repeat("─", pad)
						}
						b.WriteString(headerStyle.Render(headerText) + "\n")
					}
				}
			}

			selected := i == m.cursor

			prefix := "  "
			if selected {
				prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#bd93f9")).Bold(true).Render("▶ ")
			}

			depth := m.depth(t.ID)
			indent := ""
			if depth > 0 {
				indent = treeStyle.Render(strings.Repeat("  ", depth-1) + "  └─ ")
			}

			check := "[ ]"
			if t.Done {
				check = "[" + checkStyle.Render("✓") + "]"
			}

			titleContent := t.Title

			var renderedLine string
			if t.Done {
				renderedLine = doneStyle.Render(check + " " + titleContent)
			} else if selected {
				renderedLine = selectedStyle.Render(check + " " + titleContent)
			} else {
				renderedLine = normalStyle.Render(check + " " + titleContent)
			}

			line := prefix + indent + renderedLine

			// Truncate logic to prevent UI wrapping/breaking
			maxWidth := m.width
			var truncatedLine string
			if maxWidth > 10 {
				truncatedLine = truncate.StringWithTail(line, uint(maxWidth-4), "…")
			} else {
				truncatedLine = line
			}

			b.WriteString(truncatedLine + "\n")
		}
		body = b.String()
	}

	base := header + body + "\n" + m.footer()

	// Command bar (when active)
	if m.mode == modeCommand {
		base += "\n" + m.commandBar()
	}

	return base
}

func (m Model) footer() string {
	viewMode := "open"
	if m.showDone {
		viewMode = "all"
	}
	left := fmt.Sprintf(" j/k:nav  J/K:reorder  H/L:indent  a:add  s:sub  g:heading  d:del  space:✓  t:%s  ?:help  ::cmd  q:quit ", viewMode)

	footerBar := lipgloss.NewStyle().
		Background(lipgloss.Color("#6272A4")).
		Foreground(lipgloss.Color("#F8F8F2")).
		Bold(true).
		Render(left)

	status := ""
	if m.status != "" && time.Since(m.statusSet) < 3*time.Second {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")).Render("  " + m.status)
	}

	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("error: " + m.err.Error())
	}

	pathLine := lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#6272A4")).Render("⟫ " + m.store.Path)

	return "\n" + footerBar + status + "\n" + pathLine
}

func (m Model) commandBar() string {
	box := lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8"))

	return box.Render(m.cmdInput.View())
}

func (m Model) helpModalView() string {
	helpTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6")).Render("gotodo help")
	section := func(name string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8BE9FD")).Render(name)
	}
	key := func(k string) string {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F1FA8C")).Render(k)
	}

	content := strings.Join([]string{
		helpTitle,
		"",
		section("Navigation"),
		"  " + key("j/k") + " or " + key("↑/↓") + "     move cursor",
		"  " + key("J/K") + "             reorder task up/down",
		"  " + key("H/L") + "             promote / demote (indent)",
		"",
		section("Tasks"),
		"  " + key("a") + "               add task  (use +tag for project)",
		"  " + key("s") + "               add subtask under cursor",
		"  " + key("g") + "               add heading / group",
		"  " + key("space") + "           toggle done",
		"  " + key("d") + " / " + key("backspace") + "   delete task",
		"  " + key("t") + "               toggle show done / open",
		"",
		section("Meta"),
		"  " + key(":") + "               command bar",
		"  " + key("?") + "               this help",
		"  " + key("q") + " / " + key("ctrl+c") + "     quit",
		"",
		section("Commands  (: bar)"),
		"  " + key(":w") + "              save",
		"  " + key(":q") + "              quit",
		"  " + key(":toggle done") + "    toggle done/all view",
		"  " + key(":set file=~/...") + " switch save file (reloads)",
		"",
		subTitleStyle.Render("close: esc / q / ?"),
	}, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		Padding(1, 3).
		BorderForeground(lipgloss.Color("#bd93f9")).
		Render(content)

	if m.width <= 0 || m.height <= 0 {
		return modal
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}