package app

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"taskg/internal/styles"
	"taskg/internal/taskmeta"
	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model: TaskModel represents the TUI state for browsing Taskfile tasks.

// TaskModel is the refactored model focused on Taskfile tasks.
type TaskModel struct {
	tasks         []taskmeta.Task
	originalTasks []taskmeta.Task // to preserve original order
	filteredTasks []taskmeta.Task
	selected      int
	searchMode    bool
	searchQuery   string
	searchInput   textinput.Model
	theme         styles.Theme
	mouseEnabled  bool
	width         int
	height        int
	lastCommand   string
	statusMessage string
	statusTimeout time.Time
	projectName   string
	projectRoot   string // for refresh functionality
	errorMessage  string
	// favorites placeholders
	favorites       map[string]bool
	quitAfterSelect bool
	// tab scroll state
	tabOffset int // index of first visible tab
	// header indent (logo width + gap) used to align tabs under title
	headerIndent int
	// vertical scroll state
	listOffset int
	// cached dynamic measurements
	itemHeight int // includes trailing spacing newline after each item
	// tab-related state
	tabs         []string          // list of tab names (prefixes + "main")
	activeTab    string           // currently active tab name
	tabTasks     map[string][]taskmeta.Task // tasks grouped by tab
	sortMode     string           // "file" or "alpha"
}


type tickMsg time.Time


// refreshMsg is sent when task refresh is complete
type refreshMsg struct {
	tasks []taskmeta.Task
	err   error
}

func NewTaskModel(tasks []taskmeta.Task, themeName string, mouseEnabled bool, projectName string) *TaskModel {
	theme := styles.NewDarkTheme()
	if themeName == "light" {
		theme = styles.NewLightTheme()
	}

	// Sort tasks by line number to preserve order from Taskfile
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Line < tasks[j].Line
	})

	// Make a copy of the original tasks to restore sorting
	originalTasks := make([]taskmeta.Task, len(tasks))
	copy(originalTasks, tasks)

	m := &TaskModel{
		tasks:         tasks,
		originalTasks: originalTasks,
		filteredTasks: tasks,
		theme:         theme,
		mouseEnabled:  mouseEnabled,
		statusTimeout: time.Now(),
		projectName:   projectName,
		favorites:     make(map[string]bool),
		tabTasks:      make(map[string][]taskmeta.Task),
		sortMode:      "file", // default to file order
	}
	ti := textinput.New()
	ti.Placeholder = "Type to filter tasks"
	ti.CharLimit = 128
	ti.Width = 40
	ti.Prompt = "🔍 "
	m.searchInput = ti
	m.buildTabs()  // Build tabs from tasks
	m.updateFilter() // Apply initial filter
	return m
}

// Error sets a persistent empty-state error message.
func (m *TaskModel) Error(msg string) { m.errorMessage = msg }

// SetProjectRoot sets the project root for refresh functionality
func (m *TaskModel) SetProjectRoot(root string) { m.projectRoot = root }

func (m TaskModel) Init() tea.Cmd { return tickCmd() }
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *TaskModel) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		if m.projectRoot == "" {
			return refreshMsg{nil, fmt.Errorf("no project root set")}
		}
		tasks, err := taskmeta.DiscoverTasks(m.projectRoot)
		return refreshMsg{tasks, err}
	}
}

func (m *TaskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureSelectionVisible()
	case tea.KeyMsg:
		return m.handleKeys(msg)
	case tea.MouseMsg:
		if !m.mouseEnabled {
			return m, nil
		}
		return m.handleMouse(msg)
	case tickMsg:
		return m, tickCmd()
	case refreshMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("Refresh failed: %v", msg.err))
		} else {
			m.tasks = msg.tasks
			m.buildTabs()  // Rebuild tabs after refresh
			m.updateFilter()
			m.setStatus(fmt.Sprintf("Refreshed - %d tasks found", len(msg.tasks)))
		}
		return m, nil
	}
	return m, nil
}

func (m *TaskModel) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchMode {
		// Handle navigation keys while in search mode so arrow keys still move
		// the selection. If it's not a navigation key, pass it to the text
		// input component for normal editing.
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
				m.ensureSelectionVisible()
			}
			return m, nil
		case "down", "j":
			if m.selected < len(m.filteredTasks)-1 {
				m.selected++
				m.ensureSelectionVisible()
			}
			return m, nil
		case "pgup":
			step := m.visibleListHeight()
			m.selected = max(0, m.selected-step)
			m.ensureSelectionVisible()
			return m, nil
		case "pgdown":
			step := m.visibleListHeight()
			m.selected = min(len(m.filteredTasks)-1, m.selected+step)
			m.ensureSelectionVisible()
			return m, nil
		case "home":
			m.selected = 0
			m.ensureSelectionVisible()
			return m, nil
		case "end":
			m.selected = len(m.filteredTasks) - 1
			m.ensureSelectionVisible()
			return m, nil
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.searchQuery = m.searchInput.Value()
		m.updateFilter()
		if msg.String() == "esc" {
			m.searchMode = false
			m.searchInput.Reset()
			m.searchQuery = ""
			m.updateFilter()
		}
		if msg.String() == "enter" {
			m.searchMode = false
			// If there are filtered tasks, execute the selected one
			if len(m.filteredTasks) > 0 {
				return m, m.markForExecution()
			}
		}
		return m, cmd
	}

	// Auto-activate search mode when the user types a printable character
	// that is not already a single-key command (navigation or quit).
	// This enables "type-to-search" UX.
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		// Reserved single-letter keys we don't want to hijack for search.
		// q: quit, j/k: navigation, r: refresh.
		if r != 'q' && r != 'j' && r != 'k' && r != 'r' && unicode.IsPrint(r) && !unicode.IsSpace(r) {
			m.searchMode = true
			m.searchInput.Focus()
			m.searchInput.SetValue(string(r))
			m.searchQuery = m.searchInput.Value()
			m.updateFilter()
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+s":
		m.toggleSortMode()
		m.setStatus(fmt.Sprintf("Sorted by %s", m.sortMode))
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r", "ctrl+r":
		// Start refresh operation
		m.setStatus("Refreshing tasks...")
		return m, m.refreshCmd()
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.ensureSelectionVisible()
		}
	case "down", "j":
		if m.selected < len(m.filteredTasks)-1 {
			m.selected++
			m.ensureSelectionVisible()
		}
	case "pgup":
		step := m.visibleListHeight()
		m.selected = max(0, m.selected-step)
		m.ensureSelectionVisible()
	case "pgdown":
		step := m.visibleListHeight()
		m.selected = min(len(m.filteredTasks)-1, m.selected+step)
		m.ensureSelectionVisible()
	case "home":
		m.selected = 0
		m.ensureSelectionVisible()
	case "end":
		m.selected = len(m.filteredTasks) - 1
		m.ensureSelectionVisible()
	case "enter":
		return m, m.markForExecution()
	case "/":
		m.searchMode = true
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.searchQuery = ""
	case "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.updateFilter()
		} else {
			// If no search query to clear, quit the app
			return m, tea.Quit
		}
	case "tab":
		// Move to next tab
		if len(m.tabs) > 1 {
			m.moveToNextTab()
		}
	case "shift+tab":
		// Move to previous tab
		if len(m.tabs) > 1 {
			m.moveToPrevTab()
		}
	case "left":
		// Move to previous tab
		if len(m.tabs) > 1 {
			m.moveToPrevTab()
		}
	case "right":
		// Move to next tab
		if len(m.tabs) > 1 {
			m.moveToNextTab()
		}
	}
	return m, nil
}

// Legacy view handlers removed.

func (m *TaskModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseLeft:
		// Check if click is on tabs (line 2, after header)
		if msg.Y == 2 && len(m.tabs) > 1 {
			// Calculate which tab was clicked
			tabIndex := m.getTabIndexAtX(msg.X)
			if tabIndex >= 0 && tabIndex < len(m.tabs) {
				m.activeTab = m.tabs[tabIndex]
				m.updateFilter()
			}
		} else if msg.Y >= 4 { // after header, tabs, and search (if present)
			adjustY := msg.Y - 4
			if m.searchMode || m.searchQuery != "" {
				adjustY-- // Account for search box
			}
			if adjustY >= 0 && adjustY < len(m.filteredTasks) {
				m.selected = adjustY
			}
		}
	case tea.MouseLeft | tea.MouseMotion:
		if msg.Y >= 4 {
			adjustY := msg.Y - 4
			if m.searchMode || m.searchQuery != "" {
				adjustY--
			}
			if adjustY >= 0 && adjustY < len(m.filteredTasks) && adjustY == m.selected {
				return m, m.markForExecution()
			}
		}
	}
	return m, nil
}

func (m *TaskModel) markForExecution() tea.Cmd {
	if len(m.filteredTasks) == 0 {
		return nil
	}
	task := m.filteredTasks[m.selected]
	m.lastCommand = task.Name
	m.quitAfterSelect = true
	return tea.Quit
}

func (m *TaskModel) toggleSortMode() {
	// Preserve selection
	var selectedTaskName string
	if len(m.filteredTasks) > 0 && m.selected >= 0 && m.selected < len(m.filteredTasks) {
		selectedTaskName = m.filteredTasks[m.selected].Name
	}

	if m.sortMode == "file" {
		m.sortMode = "alpha"
	} else {
		m.sortMode = "file"
	}

	m.buildTabs()
	m.updateFilter()

	// Restore selection
	if selectedTaskName != "" {
		for i, t := range m.filteredTasks {
			if t.Name == selectedTaskName {
				m.selected = i
				break
			}
		}
	}
	m.ensureSelectionVisible()
}

// Accessors used by main program after TUI exits.
func (m TaskModel) ShouldRun() bool   { return m.quitAfterSelect && m.lastCommand != "" }
func (m TaskModel) TaskToRun() string { return m.lastCommand }

// (Removed legacy grouping functions & types)

func (m *TaskModel) updateFilter() {
	// If there's a search query, run the search across all tasks (global
	// search), otherwise show tasks for the currently active tab.
	var baseTasks []taskmeta.Task
	if m.searchQuery != "" {
		// global search across all discovered tasks
		baseTasks = m.tasks
	} else {
		baseTasks = m.tabTasks[m.activeTab]
		if baseTasks == nil {
			baseTasks = []taskmeta.Task{}
		}
	}

	if m.searchQuery == "" {
		m.filteredTasks = baseTasks
	} else {
		q := strings.ToLower(m.searchQuery)
		var res []taskmeta.Task
		for _, t := range baseTasks {
			hay := strings.ToLower(t.Name + " " + t.Desc + " " + strings.Join(t.Cmds, " "))
			if strings.Contains(hay, q) {
				res = append(res, t)
			}
		}
		m.filteredTasks = res
	}

	if m.selected >= len(m.filteredTasks) {
		m.selected = max(0, len(m.filteredTasks)-1)
	}
	m.ensureSelectionVisible()
}

func (m *TaskModel) buildTabs() {
	prefixMap := make(map[string][]taskmeta.Task)
	var prefixes []string
	prefixSet := make(map[string]bool)

	// Use originalTasks to ensure file order is always the base
	tasksToProcess := m.originalTasks

	for _, task := range tasksToProcess {
		var prefix string
		parts := strings.SplitN(task.Name, "-", 2)
		if len(parts) > 1 {
			prefix = parts[0]
		} else {
			prefix = "main"
		}

		if !prefixSet[prefix] {
			prefixes = append(prefixes, prefix)
			prefixSet[prefix] = true
		}
		prefixMap[prefix] = append(prefixMap[prefix], task)
	}

	// Sort tasks within each tab
	for _, tasks := range prefixMap {
		if m.sortMode == "alpha" {
			sort.SliceStable(tasks, func(i, j int) bool {
				return tasks[i].Name < tasks[j].Name
			})
		} else { // "file"
			sort.SliceStable(tasks, func(i, j int) bool {
				return tasks[i].Line < tasks[j].Line
			})
		}
	}

	// Sort tabs if in alpha mode, but keep "main" first
	if m.sortMode == "alpha" {
		sort.Strings(prefixes)
	}

	// Ensure "main" tab is always first if it exists
	mainIndex := -1
	for i, p := range prefixes {
		if p == "main" {
			mainIndex = i
			break
		}
	}
	if mainIndex > 0 { // if main is not already at the start
		mainPrefix := prefixes[mainIndex]
		prefixes = append(prefixes[:mainIndex], prefixes[mainIndex+1:]...)
		prefixes = append([]string{mainPrefix}, prefixes...)
	}

	m.tabs = prefixes
	m.tabTasks = prefixMap

	// Ensure active tab is still valid
	foundActive := false
	for _, t := range m.tabs {
		if t == m.activeTab {
			foundActive = true
			break
		}
	}
	if !foundActive && len(m.tabs) > 0 {
		m.activeTab = m.tabs[0]
	}
}



func (m *TaskModel) moveToNextTab() {
	if len(m.tabs) <= 1 {
		return
	}

	// (legacy tab state save removed)

	// Find current tab index and move to next
	currentIndex := -1
	for i, tab := range m.tabs {
		if tab == m.activeTab {
			currentIndex = i
			break
		}
	}

	if currentIndex >= 0 {
		// Move to next tab only if we're not already at the last tab. Do not wrap-around.
		if currentIndex < len(m.tabs)-1 {
			nextIndex := currentIndex + 1
			m.activeTab = m.tabs[nextIndex]

			// Adjust tab offset if needed to keep new tab visible
			m.ensureTabVisible(nextIndex)
		}
	}

	// (legacy tab state restore removed)
	m.updateFilter()
}

func (m *TaskModel) moveToPrevTab() {
	if len(m.tabs) <= 1 {
		return
	}

	// (legacy tab state save removed)

	// Find current tab index and move to previous
	currentIndex := -1
	for i, tab := range m.tabs {
		if tab == m.activeTab {
			currentIndex = i
			break
		}
	}

	if currentIndex >= 0 {
		// Move to previous tab only if we're not already at the first tab. Do not wrap-around.
		if currentIndex > 0 {
			prevIndex := currentIndex - 1
			m.activeTab = m.tabs[prevIndex]

			// Adjust tab offset if needed to keep new tab visible
			m.ensureTabVisible(prevIndex)
		}
	}

	// (legacy tab state restore removed)
	m.updateFilter()
}

// (tab state persistence removed)

func (m *TaskModel) ensureTabVisible(tabIndex int) {
	if len(m.tabs) <= 1 {
		return
	}

	// Calculate how many tabs can fit in current width
	availableWidth := m.width - 14 // Leave space for borders and arrows
	if availableWidth < 20 {
		availableWidth = 20
	}

	visibleCount := 0
	currentWidth := 0

	for i := 0; i < len(m.tabs); i++ {
		tab := m.tabs[i]
		tabName := tab
		if tab == "main" {
			tabName = "Main"
		} else {
			tabName = m.titleCase(tab)
		}

		// Account for highlight bar and space (2 chars) + padding + margins
		tabWidth := len(tabName) + 8 // highlight bar + space + padding + margins
		if currentWidth + tabWidth > availableWidth {
			break
		}
		currentWidth += tabWidth
		visibleCount++
	}

	if visibleCount == 0 {
		visibleCount = 1
	}

	// Adjust offset to make sure the target tab is visible
	if tabIndex < m.tabOffset {
		m.tabOffset = tabIndex
	} else if tabIndex >= m.tabOffset + visibleCount {
		m.tabOffset = tabIndex - visibleCount + 1
	}

	// Ensure offset is within valid range
	maxOffset := len(m.tabs) - visibleCount
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.tabOffset > maxOffset {
		m.tabOffset = maxOffset
	}
	if m.tabOffset < 0 {
		m.tabOffset = 0
	}
}

func (m *TaskModel) getTabIndexAtX(x int) int {
	if len(m.tabs) <= 1 {
		return -1
	}

	// Simple approximation - each tab takes about 10-15 characters
	// This is a rough estimate, for precise clicking we'd need to track exact positions
	// Start after border padding plus header indent so clicks map when tabs are indented under the title/logo.
	pos := 2 + m.headerIndent
	for i := m.tabOffset; i < len(m.tabs); i++ {
		tab := m.tabs[i]
		tabWidth := len(tab) + 8 // tab name + highlight bar + space + padding + margins
		if x >= pos && x < pos+tabWidth {
			return i
		}
		pos += tabWidth
	}
	return -1
}

func (m *TaskModel) setStatus(message string) {
	m.statusMessage = message
	m.statusTimeout = time.Now().Add(3 * time.Second)
}

// visibleListHeight calculates how many command boxes fit given current height.
// Layout rows: 1 title + 1 tabs (if any) + 1 search (optional) + list + 1 status + 1 footer borders/padding already handled by container.
func (m *TaskModel) visibleListHeight() int {
	// Dynamically measure one item (including spacing newline) the first time.
	if m.itemHeight == 0 {
		m.itemHeight = m.measureItemHeight()
		if m.itemHeight <= 0 {
			m.itemHeight = 7
		} // sane fallback
	}

	const (
		containerOverhead = 4 // AppContainer border + padding vertical
		headerHeight      = 2
		tabsHeight        = 3
		searchHeight      = 3
		statusHeight      = 1
		footerHeight      = 3
	)
	avail := m.height
	if avail <= 0 {
		avail = 24
	}
	inner := avail - containerOverhead
	if inner < 10 {
		inner = 10
	}
	overhead := headerHeight + statusHeight + footerHeight
	// Add tabs height if we have multiple tabs
	if len(m.tabs) > 1 {
		overhead += tabsHeight
	}
	if m.searchMode || m.searchQuery != "" {
		overhead += searchHeight
	}
	remaining := inner - overhead
	if remaining < m.itemHeight {
		return 1
	}
	items := remaining / m.itemHeight
	if items < 1 {
		items = 1
	}
	return items
}

// measureItemHeight renders a representative command box and counts lines.
func (m *TaskModel) measureItemHeight() int {
	// Need inner width similar to renderList
	termWidth := m.width
	if termWidth <= 0 {
		termWidth = 100
	}
	// Determine container inner width dynamically from AppContainer frame size
	appFrameW, _ := m.theme.AppContainer.GetFrameSize()
	innerWidth := termWidth - appFrameW
	if innerWidth < 40 {
		innerWidth = 40
	}
	// sample multi-line format (task + commands)
	sampleTask := "  • sample-task - Sample description"
	sampleCmd := "    [echo hello | ls -la]"
	sampleContent := sampleTask + "\n" + sampleCmd

	style := m.theme.CommandBox
	str := style.Copy().Width(innerWidth).Render(sampleContent)
	// Add the spacing newline we append after every item in list rendering.
	str += "\n"
	lines := strings.Count(str, "\n")
	return lines
}

// ensureSelectionVisible adjusts listOffset to keep selected index in viewport.
func (m *TaskModel) ensureSelectionVisible() {
	listHeight := m.visibleListHeight()
	if m.selected < m.listOffset {
		m.listOffset = m.selected
	}
	if m.selected >= m.listOffset+listHeight {
		m.listOffset = m.selected - listHeight + 1
	}
	maxOffset := max(0, len(m.filteredTasks)-listHeight)
	if m.listOffset > maxOffset {
		m.listOffset = maxOffset
	}
	if m.listOffset < 0 {
		m.listOffset = 0
	}
}

func (m TaskModel) View() string { return m.renderList() }

func (m TaskModel) renderTabs(width int) string {
	if len(m.tabs) <= 1 {
		return ""
	}

	// tabParts removed; we build renderedTabs and then truncate/join below

	// We'll build the tab pieces (without arrows), then ensure the final
	// output fits on a single line by truncating the tab content if needed.
	// Reserve a small amount of space for left/right arrows when present so
	// the arrows are always visible and tabs never wrap to multiple lines.

	// Calculate available width for tabs and reserve for borders/padding
	availableWidth := width - 11 // small margin for arrows/borders
	if availableWidth < 20 {
		availableWidth = 20
	}

	// Render tab parts (no arrows yet)
	var renderedTabs []string
	for i := m.tabOffset; i < len(m.tabs); i++ {
		tab := m.tabs[i]
		tabName := tab
		if tab == "main" {
			tabName = "Main"
		} else {
			tabName = m.titleCase(tab)
		}

		if tab == m.activeTab {
			// Add vertical bar highlight for active tab
			highlightBar := m.theme.Highlight.Render("▎")
			tabContent := highlightBar + " " + tabName
			renderedTabs = append(renderedTabs, m.theme.TabActive.Render(tabContent))
		} else {
			// Add spaces to align with active tab (bar + space == 2 chars)
			tabContent := "  " + tabName
			renderedTabs = append(renderedTabs, m.theme.TabInactive.Render(tabContent))
		}
	}

	// Join without arrows to measure width
	tabsContent := strings.Join(renderedTabs, "")

	// Determine whether arrows will be needed
	leftArrow := ""
	rightArrow := ""
	if m.tabOffset > 0 {
		leftArrow = m.theme.TabArrow.Render("◀")
	}
	// A simple heuristic: if there are tabs beyond the last we attempted to render
	// then show the right arrow. We can approximate this by checking if the raw
	// rendered width exceeds the available space.
	// Reserve space for arrows when truncating so they remain visible.
	reservedForArrows := 0
	if leftArrow != "" {
		reservedForArrows += lipgloss.Width(leftArrow)
	}

	// If raw content would overflow availableWidth, we'll reserve space for a right arrow
	if lipgloss.Width(tabsContent)+reservedForArrows > availableWidth {
		rightArrow = m.theme.TabArrow.Render("▶")
		reservedForArrows += lipgloss.Width(rightArrow)
	}

	// Compute content width available for tab text (avoid negative)
	contentWidth := availableWidth - reservedForArrows
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Truncate the joined tabs content to fit into the single-line area.
	// This prevents wrapping. We keep the left/right arrows outside the
	// truncated content so they're always visible.
	truncated := truncateStringToWidth(tabsContent, contentWidth)

	// Compose final tab line with arrows and truncated content
	finalTabs := leftArrow + truncated + rightArrow

	return m.theme.TabsContainer.Copy().Width(width).Render(finalTabs)
}

func (m TaskModel) renderList() string {
	var content strings.Builder

	// Determine terminal width.
	termWidth := int(float64(m.width) * 0.98)
	if termWidth <= 0 {
		termWidth = 98 // fallback
	}

	// Determine inner usable width inside AppContainer borders/padding.
	appFrameW, _ := m.theme.AppContainer.GetFrameSize()
	innerWidth := termWidth - appFrameW
	if innerWidth < 40 { // sensible minimum
		innerWidth = 40
	}

	// Refactored header: title on the left, logo on the far right (two lines).
	proj := m.projectName
	if proj == "" { proj = "(no Taskfile)" }
	appTitle := "Task Runner Gui - taskg" // could append proj if desired
	secondLine := "" // reserved for future help/hints

	// Logo (2-line block glyph) now rendered at the right edge
	logoLines := []string{"░▀░▀░  ", "░▄░▄░"}
	logoStyledLines := make([]string, len(logoLines))
	logoWidth := 0
	for i, l := range logoLines {
		logoStyledLines[i] = m.theme.Logo.Copy().Render(l)
		if w := lipgloss.Width(l); w > logoWidth { logoWidth = w }
	}

	// Render title/help left; compute padding so logo aligns right.
	// We ignore any previous left indent (logo moved to right) so tabs start at col 0.
	m.headerIndent = 0

	titleRendered := m.theme.AppTitle.Render(appTitle)
	secondRendered := m.theme.Help.Render(secondLine)

	space1 := innerWidth - lipgloss.Width(titleRendered) - logoWidth
	if space1 < 1 { space1 = 1 }
	space2 := innerWidth - lipgloss.Width(secondRendered) - logoWidth
	if space2 < 1 { space2 = 1 }

	firstLine := titleRendered + strings.Repeat(" ", space1) + logoStyledLines[0]
	secondLineOut := secondRendered + strings.Repeat(" ", space2) + logoStyledLines[1]
	content.WriteString(firstLine + "\n" + secondLineOut + "\n")

	// Render tabs if we have multiple tabs. We indent them so the first tab aligns
	// with the title (which starts after the logo). headerIndent is stored for
	// mouse hit testing.
	if len(m.tabs) > 1 {
		// Header indent no longer needed (logo on right); keep 0 so first tab aligns with title start.
		m.headerIndent = 0
		content.WriteString(m.renderTabs(innerWidth) + "\n")
	} else { m.headerIndent = 0 }

	// Search
	if m.searchMode {
		box := m.theme.SearchBox.Copy()
		content.WriteString(box.Width(innerWidth).Render(m.searchInput.View()) + "\n")
	} else if m.searchQuery != "" {
		info := fmt.Sprintf("🔍 %s  ( / edit  esc clear )", m.searchQuery)
		box := m.theme.SearchBox.Copy()
		content.WriteString(box.Width(innerWidth).Render(info) + "\n")
	}

	if len(m.filteredTasks) == 0 {
		help := m.theme.Help.Copy()
		content.WriteString(help.Width(innerWidth).Render("No tasks found") + "\n")
	}
	if len(m.filteredTasks) == 0 && m.errorMessage != "" {
		errStyle := m.theme.Error.Copy()
		content.WriteString(errStyle.Width(innerWidth).Render(m.errorMessage) + "\n")
		help := m.theme.Help.Copy()
		content.WriteString(help.Width(innerWidth).Render("Create a Taskfile.yml, e.g:\nversion: '3'\ntasks:\n  hello:\n    desc: Say hello\n    cmds:\n      - echo 'Hello from Task'") + "\n")
	}

	// Command list window with vertical scrolling
	listHeight := m.visibleListHeight()
	if listHeight < 1 {
		listHeight = 1
	}
	// clamp listOffset in case of data shrink
	maxOffset := max(0, len(m.filteredTasks)-listHeight)
	if m.listOffset > maxOffset {
		m.listOffset = maxOffset
	}
	end := min(len(m.filteredTasks), m.listOffset+listHeight)
	for i := m.listOffset; i < end; i++ {
		t := m.filteredTasks[i]
		// Multi-line format: [indicator] task-name - description
		//                    [indent] [command1 | command2 | ...]
		var prefix string
		var taskStyle lipgloss.Style
		if i == m.selected {
			bar := m.theme.Highlight.Render("▎")
			dot := m.theme.Highlight.Render("•")
			prefix = fmt.Sprintf("%s %s", bar, dot)
			taskStyle = m.theme.Highlight
		} else {
			// Two spaces replace the bar + following space (bar + space == width 2)
			dot := m.theme.Accent.Render("•")
			prefix = fmt.Sprintf("  %s", dot)
			taskStyle = m.theme.TaskName
		}

		// Format: task-name - description (if available)
		taskText := taskStyle.Render(t.Name)
		if t.Desc != "" && t.Desc != "-" {
			// Do NOT accent the description when selected; only the name gets highlight.
			descStyle := m.theme.Command
			taskText += " - " + descStyle.Render(t.Desc)
		}

		// First line: task name and description
		line := fmt.Sprintf("%s %s", prefix, taskText)

		// Second line: commands (indented)
		var cmdLine string
		if len(t.Cmds) > 0 {
			// Create indented prefix for commands
			var cmdPrefix string
			if i == m.selected {
				cmdPrefix = "    " // 4 spaces to align under the task text
			} else {
				cmdPrefix = "    " // 4 spaces to align under the task text
			}

			// Format commands with separators. Keep same style whether selected or not so only task name pops.
			cmdStyle := m.theme.Description

			// Join commands with " | " separator and wrap in brackets
			cmdText := "[" + strings.Join(t.Cmds, " | ") + "]"
			cmdLine = cmdPrefix + cmdStyle.Render(cmdText)
		}

		// Combine both lines
		var fullContent string
		if cmdLine != "" {
			fullContent = line + "\n" + cmdLine
		} else {
			fullContent = line
		}

		style := m.theme.CommandBox
		if i == m.selected { style = m.theme.SelectedWire }
		box := style.Copy()
		content.WriteString(box.Width(innerWidth).Render(fullContent) + "\n")
	}

	// After changing spacing we must recompute itemHeight if theme changed sizes.
	if m.itemHeight == 0 {
		m.itemHeight = m.measureItemHeight()
	}

	// Status line (always reserve a line to avoid layout jump)
	statusText := ""
	if time.Now().Before(m.statusTimeout) && m.statusMessage != "" {
		statusText = m.statusMessage
	}
	status := m.theme.Status.Copy()
	content.WriteString(status.Width(innerWidth).Render(statusText) + "\n")

	// Build footer parts with consistent layout
	// Order: pager | move | tab switch | enter | search | refresh | quit
	parts := []string{}

	// Add page counter first (will be added at the end, but we'll reorder)
	var pageCounter string
	if len(m.filteredTasks) > 0 {
		// Use fixed width formatting to prevent misalignment during navigation
		maxItems := len(m.filteredTasks)
		current := m.selected + 1
		// Calculate width needed for largest possible numbers
		maxWidth := len(fmt.Sprintf("%d/%d", maxItems, maxItems))
		pageStr := fmt.Sprintf("%*s", maxWidth, fmt.Sprintf("%d/%d", current, maxItems))
		pageCounter = m.theme.Highlight.Render(pageStr)
		parts = append(parts, pageCounter)
	}

	// Add move
	parts = append(parts, "↑↓ move")

	// Add tab switch (if applicable)
	if len(m.tabs) > 1 {
		parts = append(parts, "←→/Tab switch")
	}

	// Add highlighted "Enter run"
	enterRun := m.theme.Highlight.Render("Enter run")
	parts = append(parts, enterRun)

	// Add search
	parts = append(parts, "/ search")

	// Add refresh
	parts = append(parts, "r/^R refresh")

	// Add sort mode indicator
	var sortIndicator string
	if m.sortMode == "alpha" {
		sortIndicator = "Sort: A→Z (^S)"
	} else {
		sortIndicator = "Sort: Original (^S)"
	}
	parts = append(parts, sortIndicator)

	// Add quit
	parts = append(parts, "q quit")

	// Ensure footer fits within width and doesn't overflow
	separator := "  │  "
	footerContent := strings.Join(parts, separator)

	// Check if content would overflow and adjust if needed
	footerWidth := lipgloss.Width(footerContent)
	if footerWidth > innerWidth {
		// If overflow, try shorter separator
		separator = " │ "
		footerContent = strings.Join(parts, separator)
		footerWidth = lipgloss.Width(footerContent)

		// If still overflows, truncate less important parts
		if footerWidth > innerWidth && len(parts) > 4 {
			// Remove the search hint if needed to fit
			shortParts := parts[1:] // Remove "/ search"
			footerContent = strings.Join(shortParts, separator)
		}
	}

	footerBox := m.theme.FooterBox.Copy()
	footer := footerBox.Width(innerWidth).Render(footerContent)
	content.WriteString(footer)

	// Final app container: set width then render
	finalRender := m.theme.AppContainer.Copy().Width(termWidth).Render(content.String())

	// Ensure we never emit more lines than the terminal height. This keeps
	// the header at the top of the viewport and prevents the terminal from
	// scrolling the header out of view when the item list grows large or when
	// switching tabs which can change the rendered height.
	// If m.height is not known (0) or too small, fall back to returning the
	// whole render so Bubble Tea can manage it, but prefer trimming when
	// possible.
	if m.height > 0 {
		lines := strings.Split(finalRender, "\n")
		// If rendered lines exceed terminal height, keep only the top lines
		// so the header remains visible.
		if len(lines) > m.height {
			lines = lines[:m.height]
			finalRender = strings.Join(lines, "\n")
		}
	}

	return finalRender
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *TaskModel) titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// truncateStringToWidth cuts s so its visible width (measured by lipgloss.Width)
// does not exceed maxW. If truncation is required we append a single right
// ellipsis character to indicate truncation. This is a small helper because
// older lipgloss versions may not provide a Truncate helper.
func truncateStringToWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	// Reserve 1 char for ellipsis
	// Walk runes until adding one more would exceed maxW-1
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		b.WriteRune(runes[i])
		if lipgloss.Width(b.String()) >= maxW-1 {
			break
		}
	}
	res := b.String()
	// If we're still too long, trim last rune(s)
	for lipgloss.Width(res) > maxW-1 {
		res = string([]rune(res)[:len([]rune(res))-1])
	}
	return res + "…"
}
