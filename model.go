package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	pageSize = 20
)

// Focus states
type focusState int

const (
	focusQuery focusState = iota
	focusResults
	focusDetail
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	queryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	queryBoxFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF6B6B")).
				Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#5A5A5A")).
				Padding(0, 1)

	tableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#7D56F4")).
				Foreground(lipgloss.Color("#FAFAFA"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#353533")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7D56F4")).
				MarginBottom(1)

	fieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Width(20)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AFAFAF"))

	fieldInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AFAFAF"))

	fieldInputFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#3A3A5A"))

	readOnlyBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B")).
				Bold(true)

	editableBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#73F59F")).
				Bold(true)
)

// QueryResult holds the result of a SQL query
type QueryResult struct {
	Columns []string
	Rows    [][]string
	Error   error
}

// QueryMeta holds parsed metadata about the query
type QueryMeta struct {
	TableName  string
	IsEditable bool
	IDColumn   string
	IDIndex    int
}

// DetailView holds the state for the detail/edit view
type DetailView struct {
	rowIndex           int
	originalRow        []string
	inputs             []textinput.Model
	focusedField       int
	scrollOffset       int
	visibleFields      int
	contentScrollOffset int // scroll offset within a multi-line field
}

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

		// Resize query window - only works in results view (not when typing in query)
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

		// Handle detail view keys first
		if m.focus == focusDetail && m.detailView != nil {
			switch msg.String() {
			case "esc":
				// Close detail view, go back to results
				m.focus = focusResults
				m.detailView = nil
				return m, nil

			case "f5", "ctrl+u":
				// Generate UPDATE and append to query window
				if m.queryMeta != nil && m.queryMeta.IsEditable {
					updateSQL := m.generateUpdateSQL()
					if updateSQL != "" {
						m.appendQueryToTextarea(updateSQL)
						m.focus = focusQuery
						m.textarea.Focus()
						m.detailView = nil
						m.statusMessage = "UPDATE statement appended. Press Ctrl+R to execute."
						return m, nil
					}
					m.statusMessage = "No changes to update."
				}
				return m, nil

			case "f6", "ctrl+d":
				// Generate DELETE and append to query window
				if m.queryMeta != nil && m.queryMeta.IsEditable {
					deleteSQL := m.generateDeleteSQL()
					if deleteSQL != "" {
						m.appendQueryToTextarea(deleteSQL)
						m.focus = focusQuery
						m.textarea.Focus()
						m.detailView = nil
						m.statusMessage = "DELETE statement appended. Press Ctrl+R to execute."
						return m, nil
					}
				}
				return m, nil

			case "f7", "ctrl+i":
				// Generate INSERT and append to query window
				if m.queryMeta != nil && m.queryMeta.IsEditable {
					insertSQL := m.generateInsertSQL()
					if insertSQL != "" {
						m.appendQueryToTextarea(insertSQL)
						m.focus = focusQuery
						m.textarea.Focus()
						m.detailView = nil
						m.statusMessage = "INSERT statement appended. Press Ctrl+R to execute."
						return m, nil
					}
				}
				return m, nil

			case "up", "shift+tab":
				if m.detailView.focusedField > 0 {
					m.detailView.inputs[m.detailView.focusedField].Blur()
					m.detailView.focusedField--
					m.detailView.contentScrollOffset = 0 // Reset content scroll when changing fields
					m.detailView.inputs[m.detailView.focusedField].Focus()
					// Adjust scroll if needed
					if m.detailView.focusedField < m.detailView.scrollOffset {
						m.detailView.scrollOffset = m.detailView.focusedField
					}
				}
				return m, nil

			case "down", "tab":
				if m.detailView.focusedField < len(m.detailView.inputs)-1 {
					m.detailView.inputs[m.detailView.focusedField].Blur()
					m.detailView.focusedField++
					m.detailView.contentScrollOffset = 0 // Reset content scroll when changing fields
					m.detailView.inputs[m.detailView.focusedField].Focus()
					// Adjust scroll if needed
					if m.detailView.focusedField >= m.detailView.scrollOffset+m.detailView.visibleFields {
						m.detailView.scrollOffset = m.detailView.focusedField - m.detailView.visibleFields + 1
					}
				}
				return m, nil

			case "pgdown":
				// Scroll down within multi-line content
				val := m.detailView.originalRow[m.detailView.focusedField]
				if strings.Contains(val, "\n") {
					lines := strings.Split(val, "\n")
					maxScroll := len(lines) - 10 // Keep at least 10 lines visible
					if maxScroll < 0 {
						maxScroll = 0
					}
					m.detailView.contentScrollOffset += 10
					if m.detailView.contentScrollOffset > maxScroll {
						m.detailView.contentScrollOffset = maxScroll
					}
				}
				return m, nil

			case "pgup":
				// Scroll up within multi-line content
				m.detailView.contentScrollOffset -= 10
				if m.detailView.contentScrollOffset < 0 {
					m.detailView.contentScrollOffset = 0
				}
				return m, nil

			default:
				// Update the focused input
				if m.queryMeta != nil && m.queryMeta.IsEditable {
					var cmd tea.Cmd
					m.detailView.inputs[m.detailView.focusedField], cmd = m.detailView.inputs[m.detailView.focusedField].Update(msg)
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}
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
			if m.result != nil && len(m.result.Rows) > 0 {
				if m.focus == focusQuery {
					m.focus = focusResults
					m.textarea.Blur()
				} else if m.focus == focusResults {
					m.focus = focusQuery
					m.textarea.Focus()
				}
			}
			return m, nil

		case "enter":
			// Open detail view when in results
			if m.focus == focusResults && m.result != nil && len(m.result.Rows) > 0 {
				m.openDetailView()
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
				m.result = m.executeQuery(query)
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
			switch msg.String() {
			case "up", "k":
				if m.selectedRow > 0 {
					m.selectedRow--
					// Check if we need to go to previous page
					if m.selectedRow < m.currentPage*pageSize {
						m.currentPage--
					}
				}
				return m, nil

			case "down", "j":
				if m.selectedRow < len(m.result.Rows)-1 {
					m.selectedRow++
					// Check if we need to go to next page
					if m.selectedRow >= (m.currentPage+1)*pageSize {
						m.currentPage++
					}
				}
				return m, nil

			case "pgup", "ctrl+u":
				if m.currentPage > 0 {
					m.currentPage--
					m.selectedRow = m.currentPage * pageSize
				}
				return m, nil

			case "pgdown", "ctrl+d":
				if m.currentPage < m.totalPages-1 {
					m.currentPage++
					m.selectedRow = m.currentPage * pageSize
				}
				return m, nil

			case "home", "g":
				m.currentPage = 0
				m.selectedRow = 0
				return m, nil

			case "end", "G":
				m.currentPage = m.totalPages - 1
				m.selectedRow = len(m.result.Rows) - 1
				return m, nil
			}
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

	for i, val := range row {
		ti := textinput.New()
		ti.SetValue(val)
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
	}

	visibleFields := (m.height - 12) / 2
	if visibleFields < 3 {
		visibleFields = 3
	}

	m.detailView = &DetailView{
		rowIndex:      m.selectedRow,
		originalRow:   append([]string{}, row...),
		inputs:        inputs,
		focusedField:  0,
		scrollOffset:  0,
		visibleFields: visibleFields,
	}
	m.focus = focusDetail
}

// generateUpdateSQL creates an UPDATE statement from the edited fields
func (m Model) generateUpdateSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	// Get quote character based on database type
	q := m.quoteIdentifier()

	var setClauses []string
	for i, input := range m.detailView.inputs {
		newVal := input.Value()
		oldVal := m.detailView.originalRow[i]
		if newVal != oldVal {
			colName := m.result.Columns[i]
			// Escape single quotes
			escapedVal := strings.ReplaceAll(newVal, "'", "''")
			if newVal == "NULL" {
				setClauses = append(setClauses, fmt.Sprintf("%s%s%s = NULL", q, colName, q))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s%s%s = '%s'", q, colName, q, escapedVal))
			}
		}
	}

	if len(setClauses) == 0 {
		return ""
	}

	// Get the ID value
	idVal := m.detailView.originalRow[m.queryMeta.IDIndex]
	escapedID := strings.ReplaceAll(idVal, "'", "''")

	return fmt.Sprintf("UPDATE %s%s%s SET %s WHERE %s%s%s = '%s'",
		q, m.queryMeta.TableName, q,
		strings.Join(setClauses, ", "),
		q, m.queryMeta.IDColumn, q,
		escapedID)
}

// quoteIdentifier returns the identifier quote character for the database type
func (m Model) quoteIdentifier() string {
	switch m.dbType {
	case "mysql":
		return "`"
	case "postgres", "postgresql", "pg":
		return `"`
	case "sqlite", "sqlite3":
		return `"`
	default:
		return `"`
	}
}

// generateDeleteSQL creates a DELETE statement for the current row
func (m Model) generateDeleteSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	q := m.quoteIdentifier()

	// Get the ID value
	idVal := m.detailView.originalRow[m.queryMeta.IDIndex]
	escapedID := strings.ReplaceAll(idVal, "'", "''")

	return fmt.Sprintf("DELETE FROM %s%s%s WHERE %s%s%s = '%s'",
		q, m.queryMeta.TableName, q,
		q, m.queryMeta.IDColumn, q,
		escapedID)
}

// generateInsertSQL creates an INSERT statement from the current field values
func (m Model) generateInsertSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	q := m.quoteIdentifier()

	var columns []string
	var values []string

	for i, input := range m.detailView.inputs {
		colName := m.result.Columns[i]
		val := input.Value()

		// Skip the ID column for INSERT (let the database auto-generate it)
		if i == m.queryMeta.IDIndex {
			continue
		}

		columns = append(columns, fmt.Sprintf("%s%s%s", q, colName, q))

		if val == "NULL" {
			values = append(values, "NULL")
		} else {
			escapedVal := strings.ReplaceAll(val, "'", "''")
			values = append(values, fmt.Sprintf("'%s'", escapedVal))
		}
	}

	return fmt.Sprintf("INSERT INTO %s%s%s (%s) VALUES (%s)",
		q, m.queryMeta.TableName, q,
		strings.Join(columns, ", "),
		strings.Join(values, ", "))
}

// View implements tea.Model
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Show detail view if active
	if m.focus == focusDetail && m.detailView != nil {
		return m.renderDetailView()
	}

	// Calculate heights
	// Title: 1 line + 1 blank = 2
	// Query box: textarea height + 2 (border) + 1 blank = textarea.Height() + 3
	// Status bar: 1 line
	// Help: 1 line
	titleHeight := 2
	queryBoxHeight := m.textarea.Height() + 4 // includes border padding and blank line
	statusHeight := 1
	helpHeight := 1
	tableHeight := m.height - titleHeight - queryBoxHeight - statusHeight - helpHeight

	if tableHeight < 3 {
		tableHeight = 3
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🍽️  Dabble - Database Client"))
	b.WriteString("\n\n")

	// Query input
	queryStyle := queryBoxStyle
	if m.focus == focusQuery {
		queryStyle = queryBoxFocusedStyle
	}
	b.WriteString(queryStyle.Render(m.textarea.View()))
	b.WriteString("\n\n")

	// Results table area - build content then pad to fill space
	var tableContent string
	if m.result != nil {
		if m.result.Error != nil {
			tableContent = errorStyle.Render(fmt.Sprintf("Error: %v", m.result.Error))
		} else if len(m.result.Rows) > 0 {
			tableContent = m.renderTable()
		} else {
			tableContent = "Query executed successfully. No rows returned."
		}
	} else {
		tableContent = helpStyle.Render("Enter a SQL query and press Ctrl+R or F5 to execute.")
	}

	// Count lines in table content and pad to fill available space
	tableLines := strings.Count(tableContent, "\n") + 1
	b.WriteString(tableContent)

	// Pad with empty lines to push status bar to bottom
	for i := tableLines; i < tableHeight; i++ {
	b.WriteString("\n")
	}

	// Status bar
	statusText := m.statusMessage
	if m.result != nil && len(m.result.Rows) > 0 {
		editableText := ""
		if m.queryMeta != nil {
			if m.queryMeta.IsEditable {
				editableText = " [Editable]"
			} else {
				editableText = " [Read-only]"
			}
		}
		statusText = fmt.Sprintf("%s%s | Page %d/%d | Row %d/%d",
			m.statusMessage, editableText, m.currentPage+1, m.totalPages, m.selectedRow+1, len(m.result.Rows))
	}
	b.WriteString(statusBarStyle.Width(m.width).Render(statusText))
	b.WriteString("\n")

	// Help
	helpText := "Ctrl+R: Execute | Tab: Focus | ↑↓: Navigate | Enter: Detail | -/+: Resize | Ctrl+Q: Quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// renderDetailView renders the detail/edit view for a row
func (m Model) renderDetailView() string {
	// Calculate heights
	// Title: 1 line + 1 blank = 2
	// Detail header: 1 line + 1 blank = 2
	// Status bar: 1 line
	// Help: 1 line
	titleHeight := 2
	headerHeight := 2
	statusHeight := 1
	helpHeight := 1
	contentHeight := m.height - titleHeight - headerHeight - statusHeight - helpHeight

	if contentHeight < 5 {
		contentHeight = 5
	}

	// Update visible fields based on available height
	// Each field takes ~2 lines (label + input with border)
	m.detailView.visibleFields = contentHeight / 2
	if m.detailView.visibleFields < 3 {
		m.detailView.visibleFields = 3
	}
	if m.detailView.visibleFields > len(m.result.Columns) {
		m.detailView.visibleFields = len(m.result.Columns)
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🍽️  Dabble - Database Client"))
	b.WriteString("\n\n")

	// Detail view header
	editableStatus := ""
	if m.queryMeta != nil && m.queryMeta.IsEditable {
		editableStatus = editableBadgeStyle.Render(" [EDITABLE]")
	} else {
		editableStatus = readOnlyBadgeStyle.Render(" [READ-ONLY]")
	}
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Row Detail - Row %d%s", m.detailView.rowIndex+1, editableStatus)))
	b.WriteString("\n\n")

	// Fields
	endIdx := m.detailView.scrollOffset + m.detailView.visibleFields
	if endIdx > len(m.result.Columns) {
		endIdx = len(m.result.Columns)
	}

	linesWritten := 0
	maxValueLines := 15 // Max lines to show for multi-line values
	for i := m.detailView.scrollOffset; i < endIdx; i++ {
		colName := m.result.Columns[i]
		label := fieldLabelStyle.Render(colName + ":")

		if m.queryMeta != nil && m.queryMeta.IsEditable {
			// Editable field - textinput has its own styling
			inputView := m.detailView.inputs[i].View()
			// Add a focus indicator
			if i == m.detailView.focusedField {
				b.WriteString(fmt.Sprintf("%s %s\n", label, inputView))
			} else {
				b.WriteString(fmt.Sprintf("%s %s\n", label, fieldInputStyle.Render(inputView)))
			}
			linesWritten++
		} else {
			// Read-only field - handle multi-line content
			val := m.detailView.originalRow[i]
			isFocused := i == m.detailView.focusedField

			if strings.Contains(val, "\n") {
				// Multi-line value - display as a block
				b.WriteString(label)
				b.WriteString("\n")
				linesWritten++

				lines := strings.Split(val, "\n")
				totalLines := len(lines)

				// Apply content scroll offset for focused field
				scrollOffset := 0
				if isFocused {
					scrollOffset = m.detailView.contentScrollOffset
					if scrollOffset > len(lines) {
						scrollOffset = 0
					}
				}

				// Limit lines shown (show more for focused field)
				maxLines := maxValueLines
				if isFocused {
					maxLines = maxValueLines * 2
				}

				// Apply scroll offset
				startLine := scrollOffset
				endLine := scrollOffset + maxLines
				if endLine > len(lines) {
					endLine = len(lines)
				}
				displayLines := lines[startLine:endLine]

				// Style for the code block
				blockStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#98C379")).
					PaddingLeft(2)

				if isFocused {
					blockStyle = blockStyle.
						Background(lipgloss.Color("#2C323C")).
						Foreground(lipgloss.Color("#E5C07B"))
				}

				for _, line := range displayLines {
					// Truncate very long lines
					if len(line) > m.width-10 {
						line = line[:m.width-13] + "..."
					}
					b.WriteString(blockStyle.Render(line))
					b.WriteString("\n")
					linesWritten++
				}

				// Show scroll position indicator
				if isFocused && (scrollOffset > 0 || endLine < totalLines) {
					remaining := totalLines - endLine
					if scrollOffset > 0 && remaining > 0 {
						b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d lines above | ↓ %d lines below (PgUp/PgDn to scroll)", scrollOffset, remaining)))
					} else if scrollOffset > 0 {
						b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d lines above (PgUp to scroll)", scrollOffset)))
					} else {
						b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more lines (PgDn to scroll)", remaining)))
					}
					b.WriteString("\n")
					linesWritten++
				} else if endLine < totalLines {
					b.WriteString(helpStyle.Render(fmt.Sprintf("  ... (%d more lines)", totalLines-endLine)))
					b.WriteString("\n")
					linesWritten++
				}
			} else {
				// Single-line value
				displayVal := val
				// Truncate if too long, show more for focused
				maxLen := 60
				if isFocused {
					maxLen = m.width - 25
				}
				if len(displayVal) > maxLen {
					displayVal = displayVal[:maxLen-3] + "..."
				}

				style := fieldValueStyle
				if isFocused {
					style = lipgloss.NewStyle().
						Foreground(lipgloss.Color("#E5C07B")).
						Background(lipgloss.Color("#2C323C"))
				}
				b.WriteString(fmt.Sprintf("%s %s\n", label, style.Render(displayVal)))
				linesWritten++
			}
		}
	}

	// Scroll indicator
	scrollIndicatorLines := 0
	if len(m.result.Columns) > m.detailView.visibleFields {
		b.WriteString(fmt.Sprintf("\n  (Showing fields %d-%d of %d)\n",
			m.detailView.scrollOffset+1, endIdx, len(m.result.Columns)))
		scrollIndicatorLines = 2
	}

	// Pad with empty lines to push status bar to bottom
	usedLines := linesWritten + scrollIndicatorLines
	for i := usedLines; i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(statusBarStyle.Width(m.width).Render(m.statusMessage))
	b.WriteString("\n")

	// Help
	var helpText string
	if m.queryMeta != nil && m.queryMeta.IsEditable {
		helpText = "↑↓/Tab: Navigate | PgUp/Dn: Scroll | Ctrl+U/D/I: UPDATE/DELETE/INSERT | Esc: Back | Ctrl+Q: Quit"
	} else {
		helpText = "↑↓/Tab: Navigate fields | PgUp/PgDn: Scroll content | Esc: Back | Ctrl+Q: Quit"
	}
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// parseQueryMeta analyzes the query to determine if it's editable
func parseQueryMeta(query string, result *QueryResult) *QueryMeta {
	if result == nil || result.Error != nil {
		return nil
	}

	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	// Must be a SELECT query
	if !strings.HasPrefix(upperQuery, "SELECT") {
		return nil
	}

	// Check for aggregation functions that make it non-editable
	aggregateFuncs := []string{"COUNT(", "SUM(", "AVG(", "MIN(", "MAX(", "GROUP_CONCAT(", "GROUP BY", "HAVING", "DISTINCT"}
	for _, agg := range aggregateFuncs {
		if strings.Contains(upperQuery, agg) {
			return &QueryMeta{IsEditable: false}
		}
	}

	// Check for JOINs
	if strings.Contains(upperQuery, " JOIN ") {
		return &QueryMeta{IsEditable: false}
	}

	// Check for subqueries
	fromIdx := strings.Index(upperQuery, " FROM ")
	if fromIdx == -1 {
		return &QueryMeta{IsEditable: false}
	}

	// Look for multiple tables (comma in FROM clause before WHERE)
	afterFrom := query[fromIdx+6:]
	whereIdx := strings.Index(strings.ToUpper(afterFrom), " WHERE ")
	tablePart := afterFrom
	if whereIdx != -1 {
		tablePart = afterFrom[:whereIdx]
	}

	// Also check for ORDER BY, LIMIT etc
	for _, keyword := range []string{" ORDER BY ", " LIMIT ", " GROUP BY "} {
		if idx := strings.Index(strings.ToUpper(tablePart), keyword); idx != -1 {
			tablePart = tablePart[:idx]
		}
	}

	tablePart = strings.TrimSpace(tablePart)

	// Check for multiple tables
	if strings.Contains(tablePart, ",") {
		return &QueryMeta{IsEditable: false}
	}

	// Extract table name (handle backticks and aliases)
	tableName := extractTableName(tablePart)
	if tableName == "" {
		return &QueryMeta{IsEditable: false}
	}

	// Check if result has an 'id' column
	idIndex := -1
	idColumn := ""
	for i, col := range result.Columns {
		colLower := strings.ToLower(col)
		if colLower == "id" {
			idIndex = i
			idColumn = col
			break
		}
	}

	if idIndex == -1 {
		return &QueryMeta{IsEditable: false}
	}

	return &QueryMeta{
		TableName:  tableName,
		IsEditable: true,
		IDColumn:   idColumn,
		IDIndex:    idIndex,
	}
}

// extractTableName extracts the table name from a FROM clause fragment
func extractTableName(tablePart string) string {
	tablePart = strings.TrimSpace(tablePart)

	// Remove backticks
	tablePart = strings.ReplaceAll(tablePart, "`", "")

	// Handle alias (e.g., "users u" or "users AS u")
	parts := strings.Fields(tablePart)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

// getQueryUnderCursor finds and returns the SQL query that contains the cursor position
func (m Model) getQueryUnderCursor() string {
	content := m.textarea.Value()
	if strings.TrimSpace(content) == "" {
		return ""
	}

	// Get cursor line (0-indexed)
	cursorLine := m.textarea.Line()

	// Split content into lines and find which query block the cursor is in
	lines := strings.Split(content, "\n")

	// Calculate the character position at the start of the cursor line
	cursorPos := 0
	for i := 0; i < cursorLine && i < len(lines); i++ {
		cursorPos += len(lines[i]) + 1 // +1 for newline
	}
	// Add some offset into the current line (middle of line is fine for finding the query)
	if cursorLine < len(lines) {
		cursorPos += len(lines[cursorLine]) / 2
	}

	// Find all semicolon positions
	var semicolonPositions []int
	for i, ch := range content {
		if ch == ';' {
			semicolonPositions = append(semicolonPositions, i)
		}
	}

	// If no semicolons, there are no complete queries
	if len(semicolonPositions) == 0 {
		return ""
	}

	// Find which query segment contains the cursor
	// Query segments are: [0, semi1], [semi1+1, semi2], [semi2+1, semi3], ...
	queryStart := 0
	for _, semiPos := range semicolonPositions {
		if cursorPos <= semiPos {
			// Cursor is within this query (from queryStart to semiPos)
			query := strings.TrimSpace(content[queryStart : semiPos+1])
			// Remove the trailing semicolon for execution
			query = strings.TrimSuffix(query, ";")
			query = strings.TrimSpace(query)
			return query
		}
		queryStart = semiPos + 1
	}

	// Cursor is after the last semicolon - check if there's an incomplete query
	// If so, return empty (no complete query under cursor)
	remaining := strings.TrimSpace(content[queryStart:])
	if remaining == "" {
		// Cursor is right after last semicolon, return the last query
		if len(semicolonPositions) > 0 {
			lastSemi := semicolonPositions[len(semicolonPositions)-1]
			prevStart := 0
			if len(semicolonPositions) > 1 {
				prevStart = semicolonPositions[len(semicolonPositions)-2] + 1
			}
			query := strings.TrimSpace(content[prevStart : lastSemi+1])
			query = strings.TrimSuffix(query, ";")
			query = strings.TrimSpace(query)
			return query
		}
	}

	// There's incomplete text after last semicolon - no complete query under cursor
	return ""
}

// appendQueryToTextarea appends a SQL statement to the textarea and moves cursor to end
func (m *Model) appendQueryToTextarea(sql string) {
	current := m.textarea.Value()
	var newContent string

	if strings.TrimSpace(current) == "" {
		newContent = sql + ";"
	} else {
		// Ensure current content ends properly
		current = strings.TrimRight(current, " \t\n")
		if !strings.HasSuffix(current, ";") {
			current += ";"
		}
		newContent = current + "\n\n" + sql + ";"
	}

	m.textarea.SetValue(newContent)
	// Move cursor to end
	m.textarea.CursorEnd()
	// Save to file
	m.saveToFile()
}

// saveToFile saves the current textarea content to the SQL file
func (m *Model) saveToFile() {
	if m.sqlFile == "" {
		return
	}
	content := m.textarea.Value()
	// Write file, ignoring errors (we don't want to crash on save failure)
	if err := os.WriteFile(m.sqlFile, []byte(content), 0644); err == nil {
		m.lastSavedContent = content
	}
}

// hasUnsavedChanges returns true if the textarea content differs from the last saved content
func (m Model) hasUnsavedChanges() bool {
	return m.textarea.Value() != m.lastSavedContent
}

// executeQuery runs the SQL query and returns results
func (m Model) executeQuery(query string) *QueryResult {
	rows, err := m.db.Query(query)
	if err != nil {
		return &QueryResult{Error: err}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return &QueryResult{Error: err}
	}

	var resultRows [][]string
	for rows.Next() {
		// Create a slice of interface{} to hold each column
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return &QueryResult{Error: err}
		}

		// Convert to strings
		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{Error: err}
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
	}
}

// renderTable renders the results as a table
func (m Model) renderTable() string {
	if m.result == nil || len(m.result.Columns) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(m.result.Columns))
	for i, col := range m.result.Columns {
		colWidths[i] = len(col)
	}

	// Get page slice
	startIdx := m.currentPage * pageSize
	endIdx := startIdx + pageSize
	if endIdx > len(m.result.Rows) {
		endIdx = len(m.result.Rows)
	}
	pageRows := m.result.Rows[startIdx:endIdx]

	// Update widths based on data (limit to reasonable max)
	maxColWidth := 40
	for _, row := range pageRows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Cap widths
	for i := range colWidths {
		if colWidths[i] > maxColWidth {
			colWidths[i] = maxColWidth
		}
	}

	var b strings.Builder

	// Header
	var headerCells []string
	for i, col := range m.result.Columns {
		cell := truncateString(col, colWidths[i])
		cell = padRight(cell, colWidths[i])
		headerCells = append(headerCells, tableHeaderStyle.Render(cell))
	}
	b.WriteString(strings.Join(headerCells, ""))
	b.WriteString("\n")

	// Separator
	var sepParts []string
	for _, w := range colWidths {
		sepParts = append(sepParts, strings.Repeat("─", w+2))
	}
	b.WriteString(strings.Join(sepParts, ""))
	b.WriteString("\n")

	// Rows
	for rowIdx, row := range pageRows {
		actualRowIdx := startIdx + rowIdx
		var cells []string
		for i, cell := range row {
			cellStr := truncateString(cell, colWidths[i])
			cellStr = padRight(cellStr, colWidths[i])
			if actualRowIdx == m.selectedRow && m.focus == focusResults {
				cells = append(cells, selectedRowStyle.Render(tableCellStyle.Render(cellStr)))
			} else {
				cells = append(cells, tableCellStyle.Render(cellStr))
			}
		}
		b.WriteString(strings.Join(cells, ""))
		b.WriteString("\n")
	}

	return b.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
