package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
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
)

// QueryResult holds the result of a SQL query
type QueryResult struct {
	Columns []string
	Rows    [][]string
	Error   error
}

// Model is the main Bubble Tea model
type Model struct {
	db            *sql.DB
	textarea      textarea.Model
	viewport      viewport.Model
	focus         focusState
	result        *QueryResult
	selectedRow   int
	currentPage   int
	totalPages    int
	width         int
	height        int
	ready         bool
	statusMessage string
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
		switch msg.String() {
		case "ctrl+c", "esc":
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
				} else {
					m.focus = focusQuery
					m.textarea.Focus()
				}
			}
			return m, nil

		case "ctrl+enter", "f5":
			// Execute query
			query := strings.TrimSpace(m.textarea.Value())
			if query != "" {
				m.result = m.executeQuery(query)
				m.selectedRow = 0
				m.currentPage = 0
				if m.result.Error != nil {
					m.statusMessage = fmt.Sprintf("Error: %v", m.result.Error)
				} else {
					m.totalPages = (len(m.result.Rows) + pageSize - 1) / pageSize
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
	}

	// Update textarea if focused
	if m.focus == focusQuery {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
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
		statusText = fmt.Sprintf("%s | Page %d/%d | Row %d/%d",
			m.statusMessage, m.currentPage+1, m.totalPages, m.selectedRow+1, len(m.result.Rows))
	}
	b.WriteString(statusBarStyle.Width(m.width).Render(statusText))
	b.WriteString("\n")

	// Help
	helpText := "Ctrl+Enter/F5: Execute | Tab: Switch focus | ↑↓/jk: Navigate | PgUp/PgDn: Page | Esc: Back/Quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
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
