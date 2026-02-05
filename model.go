package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	pageSize = 20
)

// Model is the main Bubble Tea model
type Model struct {
	// Tab management
	tabs      []*Tab
	activeTab int

	// Global UI state
	confirmingQuit bool
	viewport       viewport.Model
	focus          focusState
	width          int
	height         int
	ready          bool
	statusMessage  string
	fileDialog     *FileDialog

	// Connection management
	vaultManager     *VaultManager
	connectionPicker *ConnectionPicker // for interactive connection switching
	creatingNewTab   bool              // true when connection picker is for new tab

	// SQL directory (global default)
	sqlDir string
}

// NewTab creates a new Tab with the given connection
func NewTab(db *sql.DB, dbType string, sqlDir string, sqlFile string, initialSQL string, connectionName string, theme Theme) *Tab {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(8)
	ta.ShowLineNumbers = true
	ta.KeyMap.InsertNewline.SetEnabled(true)

	// Load initial SQL content
	if initialSQL != "" {
		ta.SetValue(initialSQL)
	}

	return &Tab{
		db:               db,
		dbType:           dbType,
		sqlDir:           sqlDir,
		sqlFile:          sqlFile,
		lastSavedContent: initialSQL,
		textarea:         ta,
		connectionName:   connectionName,
		theme:            theme,
		highlighter:      NewSQLHighlighter(theme),
	}
}

// NewModel creates a new Model with a single initial tab
func NewModel(db *sql.DB, dbType string, sqlDir string, sqlFile string, initialSQL string, vm *VaultManager, connectionName string, theme Theme) Model {
	tab := NewTab(db, dbType, sqlDir, sqlFile, initialSQL, connectionName, theme)

	return Model{
		tabs:         []*Tab{tab},
		activeTab:    0,
		focus:        focusQuery,
		vaultManager: vm,
		sqlDir:       sqlDir,
	}
}

// activeTabPtr returns a pointer to the active tab
func (m *Model) activeTabPtr() *Tab {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab]
	}
	return nil
}

// tab returns the active tab (for read-only access)
func (m Model) tab() *Tab {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab]
	}
	return nil
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// editorFinishedMsg is sent when the external editor exits
type editorFinishedMsg struct {
	err error
}

// openInExternalEditor opens the SQL file in the user's $EDITOR
func (m *Model) openInExternalEditor() tea.Cmd {
	tab := m.activeTabPtr()
	if tab == nil {
		return nil
	}

	// Save current content before opening editor
	m.saveToFile()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // fallback to vi if EDITOR not set
	}

	c := exec.Command(editor, tab.sqlFile)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	tab := m.activeTabPtr()

	switch msg := msg.(type) {
	case editorFinishedMsg:
		// External editor closed - reload the file
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Editor error: %v", msg.err)
		} else {
			m.reloadFileFromDisk()
		}
		return m, nil

	case tea.KeyMsg:
		// Handle confirm quit dialog
		if m.confirmingQuit {
			switch msg.String() {
			case "y", "Y":
				m.saveAllTabs()
				return m, tea.Quit
			case "n", "N":
				return m, tea.Quit
			case "esc":
				m.confirmingQuit = false
				m.statusMessage = "Quit cancelled"
				return m, nil
			default:
				// Ignore other keys while confirming
				return m, nil
			}
		}

		// Global quit - works from any view
		if msg.String() == "ctrl+q" || msg.String() == "ctrl+c" {
			if m.hasUnsavedChangesAnyTab() {
				m.confirmingQuit = true
				m.statusMessage = "You have unsaved changes. Save before quitting? (y/n, Esc to cancel)"
				return m, nil
			}
			return m, tea.Quit
		}

		// Global save - Ctrl+S
		if msg.String() == "ctrl+s" {
			m.saveToFile()
			if tab != nil {
				m.statusMessage = fmt.Sprintf("Saved to %s", tab.sqlFile)
			}
			return m, nil
		}

		// Global open - Ctrl+O
		if msg.String() == "ctrl+o" {
			m.openFileDialog()
			return m, nil
		}

		// Open in external editor - Ctrl+E
		if msg.String() == "ctrl+e" {
			if tab == nil || tab.sqlFile == "" {
				m.statusMessage = "No SQL file to edit"
				return m, nil
			}
			return m, m.openInExternalEditor()
		}

		// New tab - Ctrl+T
		if msg.String() == "ctrl+t" {
			if m.vaultManager != nil {
				m.creatingNewTab = true
				m.openConnectionPicker()
			} else {
				m.statusMessage = "No vault configured - add connections with -add-conn"
			}
			return m, nil
		}

		// Switch tabs - Ctrl+Tab or Ctrl+PageDown (next), Ctrl+Shift+Tab or Ctrl+PageUp (prev)
		// Also support Shift+Tab for switching when in query/results view (not in detail view where it navigates fields)
		nextTabKeys := msg.String() == "ctrl+tab" || msg.String() == "ctrl+pgdown"
		prevTabKeys := msg.String() == "ctrl+shift+tab" || msg.String() == "ctrl+pgup"

		// Allow shift+tab for tab switching only when not in detail view or other dialogs
		if msg.String() == "shift+tab" && (m.focus == focusQuery || m.focus == focusResults) && len(m.tabs) > 1 {
			prevTabKeys = true
		}

		if nextTabKeys {
			if len(m.tabs) > 1 {
				m.saveToFile() // Save current tab before switching
				m.activeTab = (m.activeTab + 1) % len(m.tabs)
				m.reloadFileFromDisk() // Reload the new tab's file
				m.statusMessage = fmt.Sprintf("Tab %d: %s", m.activeTab+1, m.tabDisplayName(m.activeTab))
			}
			return m, nil
		}
		if prevTabKeys {
			if len(m.tabs) > 1 {
				m.saveToFile() // Save current tab before switching
				m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
				m.reloadFileFromDisk() // Reload the new tab's file
				m.statusMessage = fmt.Sprintf("Tab %d: %s", m.activeTab+1, m.tabDisplayName(m.activeTab))
			}
			return m, nil
		}

		// Close tab - Ctrl+W
		if msg.String() == "ctrl+w" {
			if len(m.tabs) > 1 {
				m.closeCurrentTab()
			} else {
				m.statusMessage = "Cannot close the last tab"
			}
			return m, nil
		}

		// Handle file dialog keys
		if m.focus == focusFileDialog && m.fileDialog != nil {
			return m.handleFileDialogKeys(msg)
		}

		// Handle connection picker keys
		if (m.focus == focusConnectionPicker || m.focus == focusNewTabPicker) && m.connectionPicker != nil {
			return m.handleConnectionPickerKeys(msg)
		}

		// Open connection picker - Ctrl+P (switch connection for current tab)
		if msg.String() == "ctrl+p" {
			if m.vaultManager != nil {
				m.creatingNewTab = false
				m.openConnectionPicker()
			} else {
				m.statusMessage = "No vault configured - add connections with -add-conn"
			}
			return m, nil
		}

		// Resize query window - works in results/banner view (not when typing in query)
		if m.focus == focusResults && tab != nil {
			switch msg.String() {
			case "-":
				// Shrink query window
				h := tab.textarea.Height()
				if h > 3 {
					tab.textarea.SetHeight(h - 1)
					m.statusMessage = fmt.Sprintf("Query window: %d lines", h-1)
				}
				return m, nil
			case "+", "=":
				// Grow query window
				h := tab.textarea.Height()
				maxHeight := m.height / 2 // Max half the screen
				if maxHeight < 5 {
					maxHeight = 5
				}
				if h < maxHeight {
					tab.textarea.SetHeight(h + 1)
					m.statusMessage = fmt.Sprintf("Query window: %d lines", h+1)
				}
				return m, nil
			}
		}

		// Also handle Ctrl+R from results/banner view (execute query)
		if m.focus == focusResults && (msg.String() == "ctrl+r" || msg.String() == "f5") {
			// Switch to query, execute, handled below
			m.focus = focusQuery
			if tab != nil {
				tab.textarea.Focus()
			}
			// Fall through to handle ctrl+r below
		}

		// Handle detail view keys first
		if m.focus == focusDetail && tab != nil && tab.detailView != nil {
			return m.handleDetailViewKeys(msg)
		}

		switch msg.String() {
		case "esc":
			// Esc goes back one level, doesn't quit
			if m.focus == focusResults {
				m.focus = focusQuery
				if tab != nil {
					tab.textarea.Focus()
				}
			}
			return m, nil

		case "tab":
			// Tab toggles between query and results/banner pane
			switch m.focus {
			case focusQuery:
				m.focus = focusResults
				if tab != nil {
					tab.textarea.Blur()
				}
			case focusResults:
				m.focus = focusQuery
				if tab != nil {
					tab.textarea.Focus()
				}
			}
			return m, nil

		case "enter":
			// Open detail view when in results (not banner)
			if m.focus == focusResults && tab != nil && tab.result != nil && len(tab.result.Rows) > 0 {
				m.openDetailView()
				return m, nil
			}
			// In banner view, Enter does nothing (no detail to show)
			if m.focus == focusResults && (tab == nil || tab.result == nil) {
				return m, nil
			}

		case "ctrl+r", "f5":
			if tab == nil {
				return m, nil
			}
			// Execute the query under the cursor
			query := m.getQueryUnderCursor()
			if query == "" {
				m.statusMessage = "No query under cursor. Queries must end with ';'"
				return m, nil
			}
			tab.lastQuery = query
			tab.result = executeQuery(tab.db, query)
			tab.queryMeta = parseQueryMeta(query, tab.result)
			tab.selectedRow = 0
			tab.currentPage = 0
			// Save the SQL file after executing
			m.saveToFile()
			if tab.result.Error != nil {
				m.statusMessage = fmt.Sprintf("Error: %v", tab.result.Error)
			} else {
				tab.totalPages = (len(tab.result.Rows) + pageSize - 1) / pageSize
				if tab.totalPages == 0 {
					tab.totalPages = 1
				}
				m.statusMessage = fmt.Sprintf("Query returned %d rows", len(tab.result.Rows))
				if len(tab.result.Rows) > 0 {
					m.focus = focusResults
					tab.textarea.Blur()
				}
			}
			return m, nil
		}

		// Handle navigation in results view
		if m.focus == focusResults && tab != nil && tab.result != nil {
			return m.handleResultsNavigation(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust textarea width for all tabs
		for _, t := range m.tabs {
			t.textarea.SetWidth(msg.Width - 4)
		}

		// On first window size, set textarea to 50% of height for all tabs
		if !m.ready {
			// Calculate 50% of available height (minus chrome)
			// Chrome: title (2 lines) + tab bar (1) + status bar (1) + help (1) + borders (~4)
			chromeHeight := 9
			availableHeight := msg.Height - chromeHeight
			targetHeight := availableHeight / 2
			if targetHeight < 5 {
				targetHeight = 5
			}
			if targetHeight > 30 {
				targetHeight = 30 // reasonable max
			}
			for _, t := range m.tabs {
				t.textarea.SetHeight(targetHeight)
			}
		}

		m.ready = true

		// Initialize viewport
		headerHeight := 6 // title + tab bar + query box + status
		footerHeight := 2 // help text
		textareaHeight := 8
		if tab != nil {
			textareaHeight = tab.textarea.Height()
		}
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight-textareaHeight)
		m.viewport.YPosition = headerHeight

		// Update detail view visible fields if open
		if tab != nil && tab.detailView != nil {
			tab.detailView.visibleFields = (msg.Height - 12) / 2
			if tab.detailView.visibleFields < 3 {
				tab.detailView.visibleFields = 3
			}
		}

	case tea.MouseMsg:
		// Handle mouse clicks on tabs
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Check if click is on the tab bar (row 2, after title)
			// Title is row 0-1, tab bar is row 2
			if msg.Y == 2 && len(m.tabs) > 1 {
				clickedTab := m.getTabAtPosition(msg.X)
				if clickedTab >= 0 && clickedTab != m.activeTab && clickedTab < len(m.tabs) {
					m.saveToFile() // Save current tab before switching
					m.activeTab = clickedTab
					m.reloadFileFromDisk() // Reload the new tab's file
					m.statusMessage = fmt.Sprintf("Tab %d: %s", m.activeTab+1, m.tabDisplayName(m.activeTab))
				}
				return m, nil
			}
		}
	}

	// Update textarea if focused
	if m.focus == focusQuery && tab != nil {
		var cmd tea.Cmd
		tab.textarea, cmd = tab.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// tabDisplayName returns a display name for a tab
func (m Model) tabDisplayName(idx int) string {
	if idx < 0 || idx >= len(m.tabs) {
		return ""
	}
	tab := m.tabs[idx]
	if tab.connectionName != "" {
		return tab.connectionName
	}
	if tab.dbType != "" {
		return tab.dbType
	}
	return "untitled"
}

// getTabAtPosition returns the tab index at the given X position, or -1 if none
func (m Model) getTabAtPosition(x int) int {
	if len(m.tabs) == 0 {
		return -1
	}

	currentX := 0
	for i := range m.tabs {
		// Calculate tab label (same logic as renderTabBar)
		label := m.tabDisplayName(i)
		if len(label) > 15 {
			label = label[:12] + "..."
		}

		// Tab label format: "N: label" with padding (0, 1) = 2 extra chars
		tabLabel := fmt.Sprintf("%d: %s", i+1, label)
		tabWidth := len(tabLabel) + 2 // +2 for padding

		if x >= currentX && x < currentX+tabWidth {
			return i
		}

		currentX += tabWidth + 1 // +1 for space between tabs
	}

	return -1
}

// closeCurrentTab closes the active tab
func (m *Model) closeCurrentTab() {
	if len(m.tabs) <= 1 {
		return
	}

	tab := m.activeTabPtr()
	if tab != nil {
		// Save before closing
		m.saveToFile()
		// Close the database connection
		if tab.db != nil {
			_ = tab.db.Close()
		}
	}

	// Remove the tab
	m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)

	// Adjust active tab index
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}

	m.statusMessage = fmt.Sprintf("Tab closed. %d tab(s) open.", len(m.tabs))
}

// saveAllTabs saves all tabs' SQL files
func (m *Model) saveAllTabs() {
	for _, tab := range m.tabs {
		if tab.sqlFile != "" {
			content := tab.textarea.Value()
			if err := os.WriteFile(tab.sqlFile, []byte(content), 0644); err == nil {
				tab.lastSavedContent = content
			}
		}
	}
}

// hasUnsavedChangesAnyTab checks if any tab has unsaved changes
func (m Model) hasUnsavedChangesAnyTab() bool {
	for _, tab := range m.tabs {
		if tab.textarea.Value() != tab.lastSavedContent {
			return true
		}
	}
	return false
}

// openDetailView creates the detail view for the selected row
func (m *Model) openDetailView() {
	tab := m.activeTabPtr()
	if tab == nil || tab.result == nil || tab.selectedRow >= len(tab.result.Rows) {
		return
	}

	row := tab.result.Rows[tab.selectedRow]
	inputs := make([]textinput.Model, len(tab.result.Columns))
	isNull := make([]bool, len(tab.result.Columns))
	originalValues := make([]CellValue, len(tab.result.Columns))

	for i, cell := range row {
		ti := textinput.New()
		// For NULL values, leave the input empty but track NULL state separately
		if cell.IsNull {
			ti.SetValue("")
			isNull[i] = true
		} else {
			ti.SetValue(cell.Value)
			isNull[i] = false
		}
		ti.CharLimit = 500
		ti.Width = 50
		ti.Prompt = "│ "
		ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5A5A5A"))
		ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
		originalValues[i] = cell
	}

	visibleFields := (m.height - 12) / 2
	if visibleFields < 3 {
		visibleFields = 3
	}

	// Copy column types
	columnTypes := make([]ColumnType, len(tab.result.ColumnTypes))
	copy(columnTypes, tab.result.ColumnTypes)

	tab.detailView = &DetailView{
		rowIndex:       tab.selectedRow,
		originalValues: originalValues,
		inputs:         inputs,
		isNull:         isNull,
		columnTypes:    columnTypes,
		focusedField:   0,
		scrollOffset:   0,
		visibleFields:  visibleFields,
	}
	m.focus = focusDetail
}

// openConnectionPicker opens the connection picker/manager dialog
func (m *Model) openConnectionPicker() {
	if m.vaultManager == nil {
		m.vaultManager = NewVaultManager()
	}

	// Load config
	if err := m.vaultManager.LoadConfig(); err != nil {
		m.statusMessage = "Failed to load config: " + err.Error()
		return
	}

	m.connectionPicker = &ConnectionPicker{
		selectedIdx:  0,
		scrollOffset: 0,
	}

	if !m.vaultManager.HasVault() {
		// No vault exists - prompt to create one
		m.connectionPicker.mode = PickerModeCreateVault
	} else if !m.vaultManager.IsUnlocked() {
		// Vault exists but is locked
		m.connectionPicker.mode = PickerModeUnlock
	} else {
		// Vault is unlocked - show connections
		m.connectionPicker.connections = m.vaultManager.ListConnections()
		m.connectionPicker.mode = PickerModeList
	}

	if m.creatingNewTab {
		m.focus = focusNewTabPicker
	} else {
		m.focus = focusConnectionPicker
	}
}

// switchConnection switches the current tab to a different database connection
func (m *Model) switchConnection(name string) error {
	tab := m.activeTabPtr()
	if tab == nil {
		return fmt.Errorf("no active tab")
	}

	if m.vaultManager == nil {
		return fmt.Errorf("no vault manager")
	}

	dsn, dbType, themeName, err := m.vaultManager.GetConnection(name)
	if err != nil {
		return err
	}

	// Auto-detect type if not specified
	if dbType == "" {
		dbType = detectDBType(dsn)
	}

	driverName := getDriverName(dbType)
	if driverName == "" {
		return fmt.Errorf("unknown database type for %q", name)
	}

	// Close old connection
	if tab.db != nil {
		_ = tab.db.Close()
	}

	// Open new connection
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to ping: %w", err)
	}

	tab.db = db
	tab.dbType = dbType
	tab.connectionName = name
	tab.theme = GetTheme(themeName)
	tab.highlighter = NewSQLHighlighter(tab.theme)

	// Clear previous results
	tab.result = nil
	tab.queryMeta = nil

	return nil
}

// createNewTab creates a new tab with the given connection
func (m *Model) createNewTab(name string) error {
	if m.vaultManager == nil {
		return fmt.Errorf("no vault manager")
	}

	dsn, dbType, themeName, err := m.vaultManager.GetConnection(name)
	if err != nil {
		return err
	}

	// Auto-detect type if not specified
	if dbType == "" {
		dbType = detectDBType(dsn)
	}

	driverName := getDriverName(dbType)
	if driverName == "" {
		return fmt.Errorf("unknown database type for %q", name)
	}

	// Open new connection
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to ping: %w", err)
	}

	// Determine SQL file path for this connection
	dbName := extractDatabaseName(dsn, dbType)
	sqlFile := dbName + ".sql"
	if m.sqlDir != "" {
		sqlFile = m.sqlDir + "/" + sqlFile
	}

	// Load initial SQL content from file (if it exists)
	var initialSQL string
	if data, err := os.ReadFile(sqlFile); err == nil {
		initialSQL = string(data)
	}

	theme := GetTheme(themeName)
	tab := NewTab(db, dbType, m.sqlDir, sqlFile, initialSQL, name, theme)

	// Size the textarea to match current tabs
	if len(m.tabs) > 0 && m.tabs[0].textarea.Height() > 0 {
		tab.textarea.SetHeight(m.tabs[0].textarea.Height())
		tab.textarea.SetWidth(m.tabs[0].textarea.Width())
	}

	// Add the new tab and switch to it
	m.tabs = append(m.tabs, tab)
	m.activeTab = len(m.tabs) - 1

	return nil
}
