package main

import (
	"database/sql"
	"fmt"
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
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#5A5A5A")).
			Padding(0, 1)

	fieldInputFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#FF6B6B")).
				Padding(0, 1)

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
	rowIndex      int
	originalRow   []string
	inputs        []textinput.Model
	focusedField  int
	scrollOffset  int
	visibleFields int
}

// Model is the main Bubble Tea model
type Model struct {
	db            *sql.DB
	textarea      textarea.Model
	viewport      viewport.Model
	focus         focusState
	result        *QueryResult
	queryMeta     *QueryMeta
	lastQuery     string
	selectedRow   int
	currentPage   int
	totalPages    int
	width         int
	height        int
	ready         bool
	statusMessage string
	detailView    *DetailView
}

// NewModel creates a new Model
func NewModel(db *sql.DB) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(true)

	return Model{
		db:       db,
		textarea: ta,
		focus:    focusQuery,
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
		// Handle detail view keys first
		if m.focus == focusDetail && m.detailView != nil {
			switch msg.String() {
			case "esc":
				// Close detail view, go back to results
				m.focus = focusResults
				m.detailView = nil
				return m, nil

			case "f5":
				// Generate UPDATE and close
				if m.queryMeta != nil && m.queryMeta.IsEditable {
					updateSQL := m.generateUpdateSQL()
					if updateSQL != "" {
						m.textarea.SetValue(updateSQL)
						m.focus = focusQuery
						m.textarea.Focus()
						m.detailView = nil
						m.statusMessage = "UPDATE statement generated. Press Ctrl+Enter to execute."
						return m, nil
					}
				}
				return m, nil

			case "up", "shift+tab":
				if m.detailView.focusedField > 0 {
					m.detailView.inputs[m.detailView.focusedField].Blur()
					m.detailView.focusedField--
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
					m.detailView.inputs[m.detailView.focusedField].Focus()
					// Adjust scroll if needed
					if m.detailView.focusedField >= m.detailView.scrollOffset+m.detailView.visibleFields {
						m.detailView.scrollOffset = m.detailView.focusedField - m.detailView.visibleFields + 1
					}
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
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.focus == focusResults {
				m.focus = focusQuery
				m.textarea.Focus()
				return m, nil
			}
			return m, tea.Quit

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

		case "ctrl+enter", "f5":
			// Execute query
			query := strings.TrimSpace(m.textarea.Value())
			if query != "" {
				m.lastQuery = query
				m.result = m.executeQuery(query)
				m.queryMeta = parseQueryMeta(query, m.result)
				m.selectedRow = 0
				m.currentPage = 0
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

	var setClauses []string
	for i, input := range m.detailView.inputs {
		newVal := input.Value()
		oldVal := m.detailView.originalRow[i]
		if newVal != oldVal {
			colName := m.result.Columns[i]
			// Escape single quotes
			escapedVal := strings.ReplaceAll(newVal, "'", "''")
			if newVal == "NULL" {
				setClauses = append(setClauses, fmt.Sprintf("%s = NULL", colName))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s = '%s'", colName, escapedVal))
			}
		}
	}

	if len(setClauses) == 0 {
		return ""
	}

	// Get the ID value
	idVal := m.detailView.originalRow[m.queryMeta.IDIndex]
	escapedID := strings.ReplaceAll(idVal, "'", "''")

	return fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%s'",
		m.queryMeta.TableName,
		strings.Join(setClauses, ", "),
		m.queryMeta.IDColumn,
		escapedID)
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

	// Results table
	if m.result != nil {
		if m.result.Error != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.result.Error)))
		} else if len(m.result.Rows) > 0 {
			b.WriteString(m.renderTable())
		} else {
			b.WriteString("Query executed successfully. No rows returned.")
		}
	}
	b.WriteString("\n")

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
	helpText := "Ctrl+Enter/F5: Execute | Tab: Switch focus | ↑↓/jk: Navigate | Enter: Detail | Esc: Back/Quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// renderDetailView renders the detail/edit view for a row
func (m Model) renderDetailView() string {
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

	for i := m.detailView.scrollOffset; i < endIdx; i++ {
		colName := m.result.Columns[i]
		label := fieldLabelStyle.Render(colName + ":")

		if m.queryMeta != nil && m.queryMeta.IsEditable {
			// Editable field
			inputStyle := fieldInputStyle
			if i == m.detailView.focusedField {
				inputStyle = fieldInputFocusedStyle
			}
			b.WriteString(fmt.Sprintf("%s %s\n", label, inputStyle.Render(m.detailView.inputs[i].View())))
		} else {
			// Read-only field
			val := m.detailView.originalRow[i]
			b.WriteString(fmt.Sprintf("%s %s\n", label, fieldValueStyle.Render(val)))
		}
	}

	// Scroll indicator
	if len(m.result.Columns) > m.detailView.visibleFields {
		b.WriteString(fmt.Sprintf("\n  (Showing fields %d-%d of %d)\n",
			m.detailView.scrollOffset+1, endIdx, len(m.result.Columns)))
	}

	b.WriteString("\n")

	// Status bar
	b.WriteString(statusBarStyle.Width(m.width).Render(m.statusMessage))
	b.WriteString("\n")

	// Help
	var helpText string
	if m.queryMeta != nil && m.queryMeta.IsEditable {
		helpText = "↑↓/Tab: Navigate fields | F5: Generate UPDATE | Esc: Back to results"
	} else {
		helpText = "↑↓/Tab: Navigate fields | Esc: Back to results"
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
