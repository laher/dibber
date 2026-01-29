package main

import (
	"database/sql"
	"fmt"

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
	db               *sql.DB
	dbType           string
	sqlFile          string
	lastSavedContent string
	confirmingQuit   bool
	textarea         textarea.Model
	viewport         viewport.Model
	focus            focusState
	result           *QueryResult
	queryMeta        *QueryMeta
	lastQuery        string
	selectedRow      int
	currentPage      int
	totalPages       int
	width            int
	height           int
	ready            bool
	statusMessage    string
	detailView       *DetailView
	fileDialog       *FileDialog
}

// NewModel creates a new Model
func NewModel(db *sql.DB, dbType string, sqlFile string, initialSQL string) Model {
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

	return Model{
		db:               db,
		dbType:           dbType,
		sqlFile:          sqlFile,
		lastSavedContent: initialSQL,
		textarea:         ta,
		focus:            focusQuery,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirm quit dialog
		if m.confirmingQuit {
			switch msg.String() {
			case "y", "Y":
				m.saveToFile()
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
			if m.hasUnsavedChanges() {
				m.confirmingQuit = true
				m.statusMessage = "You have unsaved changes. Save before quitting? (y/n, Esc to cancel)"
				return m, nil
			}
			return m, tea.Quit
		}

		// Global save - Ctrl+S
		if msg.String() == "ctrl+s" {
			m.saveToFile()
			m.statusMessage = fmt.Sprintf("Saved to %s", m.sqlFile)
			return m, nil
		}

		// Global open - Ctrl+O
		if msg.String() == "ctrl+o" {
			m.openFileDialog()
			return m, nil
		}

		// Handle file dialog keys
		if m.focus == focusFileDialog && m.fileDialog != nil {
			return m.handleFileDialogKeys(msg)
		}

		// Resize query window - works in results/banner view (not when typing in query)
		if m.focus == focusResults {
			switch msg.String() {
			case "-":
				// Shrink query window
				h := m.textarea.Height()
				if h > 3 {
					m.textarea.SetHeight(h - 1)
					m.statusMessage = fmt.Sprintf("Query window: %d lines", h-1)
				}
				return m, nil
			case "+", "=":
				// Grow query window
				h := m.textarea.Height()
				maxHeight := m.height / 2 // Max half the screen
				if maxHeight < 5 {
					maxHeight = 5
				}
				if h < maxHeight {
					m.textarea.SetHeight(h + 1)
					m.statusMessage = fmt.Sprintf("Query window: %d lines", h+1)
				}
				return m, nil
			}
		}

		// Also handle Ctrl+R from results/banner view (execute query)
		if m.focus == focusResults && (msg.String() == "ctrl+r" || msg.String() == "f5") {
			// Switch to query, execute, handled below
			m.focus = focusQuery
			m.textarea.Focus()
			// Fall through to handle ctrl+r below
		}

		// Handle detail view keys first
		if m.focus == focusDetail && m.detailView != nil {
			return m.handleDetailViewKeys(msg)
		}

		switch msg.String() {
		case "esc":
			// Esc goes back one level, doesn't quit
			if m.focus == focusResults {
				m.focus = focusQuery
				m.textarea.Focus()
			}
			return m, nil

		case "tab":
			// Tab toggles between query and results/banner pane
			switch m.focus {
			case focusQuery:
				m.focus = focusResults
				m.textarea.Blur()
			case focusResults:
				m.focus = focusQuery
				m.textarea.Focus()
			}
			return m, nil

		case "enter":
			// Open detail view when in results (not banner)
			if m.focus == focusResults && m.result != nil && len(m.result.Rows) > 0 {
				m.openDetailView()
				return m, nil
			}
			// In banner view, Enter does nothing (no detail to show)
			if m.focus == focusResults && m.result == nil {
				return m, nil
			}

		case "ctrl+r", "f5":
			// Execute the query under the cursor
			query := m.getQueryUnderCursor()
			if query == "" {
				m.statusMessage = "No query under cursor. Queries must end with ';'"
				return m, nil
			}
			m.lastQuery = query
			m.result = executeQuery(m.db, query)
			m.queryMeta = parseQueryMeta(query, m.result)
			m.selectedRow = 0
			m.currentPage = 0
			// Save the SQL file after executing
			m.saveToFile()
			if m.result.Error != nil {
				m.statusMessage = fmt.Sprintf("Error: %v", m.result.Error)
			} else {
				m.totalPages = (len(m.result.Rows) + pageSize - 1) / pageSize
				if m.totalPages == 0 {
					m.totalPages = 1
				}
				m.statusMessage = fmt.Sprintf("Query returned %d rows", len(m.result.Rows))
				if len(m.result.Rows) > 0 {
					m.focus = focusResults
					m.textarea.Blur()
				}
			}
			return m, nil
		}

		// Handle navigation in results view
		if m.focus == focusResults && m.result != nil {
			return m.handleResultsNavigation(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Adjust textarea width
		m.textarea.SetWidth(msg.Width - 4)

		// Initialize viewport
		headerHeight := 5 // title + query box + status
		footerHeight := 2 // help text
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight-m.textarea.Height())
		m.viewport.YPosition = headerHeight

		// Update detail view visible fields if open
		if m.detailView != nil {
			m.detailView.visibleFields = (msg.Height - 12) / 2
			if m.detailView.visibleFields < 3 {
				m.detailView.visibleFields = 3
			}
		}
	}

	// Update textarea if focused
	if m.focus == focusQuery {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// openDetailView creates the detail view for the selected row
func (m *Model) openDetailView() {
	if m.result == nil || m.selectedRow >= len(m.result.Rows) {
		return
	}

	row := m.result.Rows[m.selectedRow]
	inputs := make([]textinput.Model, len(m.result.Columns))
	isNull := make([]bool, len(m.result.Columns))
	originalValues := make([]CellValue, len(m.result.Columns))

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
	columnTypes := make([]ColumnType, len(m.result.ColumnTypes))
	copy(columnTypes, m.result.ColumnTypes)

	m.detailView = &DetailView{
		rowIndex:       m.selectedRow,
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
