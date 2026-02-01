package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

// renderHighlightedQuery renders the query textarea content with SQL syntax highlighting
func (m Model) renderHighlightedQuery() string {
	content := m.textarea.Value()
	lines := strings.Split(content, "\n")

	// Get cursor position
	cursorLine := m.textarea.Line()
	lineInfo := m.textarea.LineInfo()
	cursorCol := lineInfo.CharOffset

	// Get textarea dimensions
	height := m.textarea.Height()
	width := m.textarea.Width()

	// Calculate scroll offset - keep cursor visible
	scrollOffset := 0
	if cursorLine >= height {
		scrollOffset = cursorLine - height + 1
	}

	// Line number width (for alignment)
	lineNumWidth := len(fmt.Sprintf("%d", len(lines)))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}

	// Calculate content width (excluding line numbers)
	contentWidth := width
	if m.textarea.ShowLineNumbers {
		contentWidth = width - lineNumWidth - 1 // -1 for space after line number
	}

	// Styles for line numbers
	lineNumStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextDim).
		Width(lineNumWidth).
		Align(lipgloss.Right)

	cursorLineNumStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		Width(lineNumWidth).
		Align(lipgloss.Right)

	// Cursor style
	cursorStyle := lipgloss.NewStyle().
		Background(m.theme.TextBright).
		Foreground(m.theme.Secondary)

	var b strings.Builder
	isFocused := m.focus == focusQuery

	// Render visible lines
	for i := scrollOffset; i < len(lines) && i < scrollOffset+height; i++ {
		line := lines[i]

		// Line number
		if m.textarea.ShowLineNumbers {
			if i == cursorLine {
				b.WriteString(cursorLineNumStyle.Render(fmt.Sprintf("%d", i+1)))
			} else {
				b.WriteString(lineNumStyle.Render(fmt.Sprintf("%d", i+1)))
			}
			b.WriteString(" ")
		}

		// Apply syntax highlighting to the line and pad to full width
		var renderedLine string
		isCursorLine := isFocused && i == cursorLine
		cursorAtEnd := isCursorLine && cursorCol >= len([]rune(line))

		if m.highlighter != nil {
			highlightedLine := m.highlighter.HighlightLine(line)

			// If this is the cursor line and we're focused, insert cursor
			if isCursorLine {
				renderedLine = m.insertCursor(line, highlightedLine, cursorCol, cursorStyle, contentWidth)
			} else {
				renderedLine = highlightedLine
			}
		} else {
			// No highlighter, render plain
			if isCursorLine {
				renderedLine = m.insertCursorPlain(line, cursorCol, cursorStyle, contentWidth)
			} else {
				renderedLine = line
			}
		}

		// Pad line to full width
		// When cursor is at end of line, we've added a cursor block, so adjust width calculation
		effectivePlainWidth := len([]rune(line))
		if cursorAtEnd {
			effectivePlainWidth++ // Account for cursor block
		}
		b.WriteString(m.padToWidthWithVisibleWidth(renderedLine, effectivePlainWidth, contentWidth))

		if i < scrollOffset+height-1 {
			b.WriteString("\n")
		}
	}

	// Pad with empty lines if content is shorter than height
	linesRendered := len(lines) - scrollOffset
	if linesRendered < 0 {
		linesRendered = 0
	}
	if linesRendered > height {
		linesRendered = height
	}
	for i := linesRendered; i < height; i++ {
		if m.textarea.ShowLineNumbers {
			b.WriteString(strings.Repeat(" ", lineNumWidth+1))
		}
		// Pad empty lines to full width
		b.WriteString(strings.Repeat(" ", contentWidth))
		if i < height-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// insertCursor inserts a cursor character into a highlighted line at the correct position
func (m Model) insertCursor(plainLine, highlightedLine string, cursorCol int, cursorStyle lipgloss.Style, maxWidth int) string {
	// If cursor is at or past end of line, append cursor block
	plainRunes := []rune(plainLine)
	if cursorCol >= len(plainRunes) {
		return m.truncateLine(highlightedLine, maxWidth-1) + cursorStyle.Render(" ")
	}

	// We need to find where in the plainLine the cursor is, then insert styling
	// This is complex because the highlighted line has ANSI codes
	// Strategy: re-highlight the parts before and after cursor separately

	beforeCursor := string(plainRunes[:cursorCol])
	cursorChar := string(plainRunes[cursorCol])
	afterCursor := string(plainRunes[cursorCol+1:])

	// Highlight each part
	var result strings.Builder
	if beforeCursor != "" {
		result.WriteString(m.highlighter.HighlightLine(beforeCursor))
	}
	result.WriteString(cursorStyle.Render(cursorChar))
	if afterCursor != "" {
		result.WriteString(m.highlighter.HighlightLine(afterCursor))
	}

	return m.truncateLine(result.String(), maxWidth)
}

// insertCursorPlain inserts a cursor into a plain (non-highlighted) line
func (m Model) insertCursorPlain(line string, cursorCol int, cursorStyle lipgloss.Style, maxWidth int) string {
	runes := []rune(line)
	if cursorCol >= len(runes) {
		return m.truncateLine(line, maxWidth-1) + cursorStyle.Render(" ")
	}

	before := string(runes[:cursorCol])
	cursorChar := string(runes[cursorCol])
	after := string(runes[cursorCol+1:])

	return m.truncateLine(before+cursorStyle.Render(cursorChar)+after, maxWidth)
}

// truncateLine truncates a line to fit within maxWidth, accounting for ANSI codes
func (m Model) truncateLine(line string, maxWidth int) string {
	// Use uniseg to properly count grapheme clusters (visible width)
	width := uniseg.StringWidth(lipgloss.NewStyle().Render(line))
	if width <= maxWidth {
		return line
	}

	// Truncate by visible width - this is approximate for styled text
	// For now, just return as-is since the query box has its own width handling
	return line
}

// padToWidthWithVisibleWidth pads a rendered line to the specified width
// using a pre-calculated visible width
func (m Model) padToWidthWithVisibleWidth(renderedLine string, visibleWidth, targetWidth int) string {
	if visibleWidth >= targetWidth {
		return renderedLine
	}

	// Add padding spaces to reach target width
	padding := targetWidth - visibleWidth
	return renderedLine + strings.Repeat(" ", padding)
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

	// Show file dialog if active
	if m.focus == focusFileDialog && m.fileDialog != nil {
		return m.renderFileDialog()
	}

	// Show connection picker if active
	if m.focus == focusConnectionPicker && m.connectionPicker != nil {
		return m.renderConnectionPicker()
	}

	// Get themed styles
	styles := m.GetStyles()

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

	// Title - show connection name and theme if using saved connection
	titleText := "🌱  Dibber - Database Client"
	if m.connectionName != "" {
		if m.theme.Name != "" && m.theme.Name != "default" {
			titleText = fmt.Sprintf("🌱  Dibber - %s (%s) [%s]", m.connectionName, m.dbType, m.theme.Name)
		} else {
			titleText = fmt.Sprintf("🌱  Dibber - %s (%s)", m.connectionName, m.dbType)
		}
	} else if m.dbType != "" {
		titleText = fmt.Sprintf("🌱  Dibber - %s", m.dbType)
	}
	b.WriteString(styles.Title.Render(titleText))
	b.WriteString("\n\n")

	// Query input with syntax highlighting
	queryBoxStyle := styles.QueryBox
	if m.focus == focusQuery {
		queryBoxStyle = styles.QueryBoxFocused
	}
	b.WriteString(queryBoxStyle.Render(m.renderHighlightedQuery()))
	b.WriteString("\n\n")

	// Results table area - build content then pad to fill space
	var tableContent string
	resultsFocused := m.focus == focusResults

	if m.result != nil {
		if m.result.Error != nil {
			tableContent = styles.Error.Render(fmt.Sprintf("Error: %v", m.result.Error))
		} else if len(m.result.Rows) > 0 {
			tableContent = m.renderTable()
		} else {
			tableContent = "Query executed successfully. No rows returned."
		}
	} else {
		tableContent = m.renderBanner()
	}

	// Add focus indicator for results/banner area
	if resultsFocused {
		focusIndicator := styles.EditableBadge.Render("▶ ")
		tableContent = focusIndicator + tableContent
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
	b.WriteString(styles.StatusBar.Width(m.width).Render(statusText))
	b.WriteString("\n")

	// Help - context-sensitive
	var helpText string
	switch m.focus {
	case focusQuery:
		helpText = "Ctrl+R: Run | Ctrl+S: Save | Ctrl+O: Open | Ctrl+P: Connections | Tab: Switch | Ctrl+Q: Quit"
	case focusResults:
		if m.result != nil && len(m.result.Rows) > 0 {
			helpText = "↑↓: Navigate | Enter: Detail | -/+: Resize | Tab: Switch | Ctrl+P: Connections | Ctrl+Q: Quit"
		} else {
			helpText = "-/+: Resize | Tab: Switch | Ctrl+R: Run | Ctrl+P: Connections | Ctrl+Q: Quit"
		}
	default:
		helpText = "Ctrl+R: Run | Ctrl+S: Save | Ctrl+P: Connections | Tab: Switch | Ctrl+Q: Quit"
	}
	b.WriteString(styles.Help.Render(helpText))

	return b.String()
}

// renderConnectionPicker renders the connection picker/manager dialog
func (m Model) renderConnectionPicker() string {
	styles := m.GetStyles()
	var b strings.Builder

	switch m.connectionPicker.mode {
	case PickerModeCreateVault:
		b.WriteString(styles.Title.Render("🔐  Create Connection Vault"))
		b.WriteString("\n\n")
		b.WriteString("No saved connections found. Create a master password to\n")
		b.WriteString("securely store your database connections.\n\n")
		b.WriteString("  Master Password (min 8 chars):\n")
		masked := strings.Repeat("•", len(m.connectionPicker.passwordInput))
		b.WriteString(fmt.Sprintf("  %s█\n", masked))
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Continue | Esc: Cancel"))

	case PickerModeConfirmVaultPassword:
		b.WriteString(styles.Title.Render("🔐  Confirm Master Password"))
		b.WriteString("\n\n")
		b.WriteString("  Confirm Password:\n")
		masked := strings.Repeat("•", len(m.connectionPicker.confirmPasswordInput))
		b.WriteString(fmt.Sprintf("  %s█\n", masked))
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Create Vault | Esc: Back"))

	case PickerModeUnlock:
		b.WriteString(styles.Title.Render("🔐  Unlock Connection Vault"))
		b.WriteString("\n\n")
		b.WriteString("  Master Password:\n")
		masked := strings.Repeat("•", len(m.connectionPicker.passwordInput))
		b.WriteString(fmt.Sprintf("  %s█\n", masked))
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Unlock | Esc: Cancel"))

	case PickerModeList:
		b.WriteString(styles.Title.Render("🔌  Connection Manager"))
		b.WriteString("\n\n")

		if len(m.connectionPicker.connections) == 0 {
			b.WriteString("  No saved connections.\n")
			b.WriteString("  Press 'a' to add your first connection.\n")
		} else {
			visibleCount := 10
			start := m.connectionPicker.scrollOffset
			end := start + visibleCount
			if end > len(m.connectionPicker.connections) {
				end = len(m.connectionPicker.connections)
			}

			for i := start; i < end; i++ {
				name := m.connectionPicker.connections[i]
				if i == m.connectionPicker.selectedIdx {
					b.WriteString(fmt.Sprintf("  ▶ %s", styles.SelectedRow.Render(name)))
				} else {
					b.WriteString(fmt.Sprintf("    %s", name))
				}
				b.WriteString("\n")
			}
		}

		m.renderPickerError(&b, styles)

		if m.connectionName != "" {
			b.WriteString(fmt.Sprintf("\n  Current: %s", m.connectionName))
		}

		b.WriteString("\n\n")
		if len(m.connectionPicker.connections) > 0 {
			b.WriteString(styles.Help.Render("↑↓: Navigate | Enter: Connect | a: Add | d: Delete | Esc: Close"))
		} else {
			b.WriteString(styles.Help.Render("a: Add Connection | Esc: Close"))
		}

	case PickerModeAddName:
		b.WriteString(styles.Title.Render("➕  Add Connection - Name"))
		b.WriteString("\n\n")
		b.WriteString("  Enter a name for this connection:\n")
		b.WriteString(fmt.Sprintf("  %s█\n", m.connectionPicker.newConnName))
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Continue | Esc: Cancel"))

	case PickerModeAddDSN:
		b.WriteString(styles.Title.Render("➕  Add Connection - DSN"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Connection: %s\n\n", m.connectionPicker.newConnName))
		b.WriteString("  Enter the database connection string (DSN):\n")
		// Show DSN masked for security
		masked := strings.Repeat("•", len(m.connectionPicker.newConnDSN))
		b.WriteString(fmt.Sprintf("  %s█\n", masked))
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("  Examples:"))
		b.WriteString("\n")
		b.WriteString("    MySQL:    user:pass@tcp(localhost:3306)/dbname\n")
		b.WriteString("    Postgres: postgres://user:pass@localhost/dbname\n")
		b.WriteString("    SQLite:   /path/to/database.db\n")
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Continue | Esc: Back"))

	case PickerModeAddType:
		b.WriteString(styles.Title.Render("➕  Add Connection - Database Type"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Connection: %s\n\n", m.connectionPicker.newConnName))
		b.WriteString("  Select database type:\n\n")

		types := []string{"mysql", "postgres", "sqlite"}
		for _, t := range types {
			if t == m.connectionPicker.newConnType {
				b.WriteString(fmt.Sprintf("  ▶ %s\n", styles.SelectedRow.Render(t)))
			} else {
				b.WriteString(fmt.Sprintf("    %s\n", t))
			}
		}

		if m.connectionPicker.newConnType != "" {
			detected := detectDBType(m.connectionPicker.newConnDSN)
			if detected != "" {
				b.WriteString(fmt.Sprintf("\n  (Auto-detected: %s)", detected))
			}
		}
		m.renderPickerError(&b, styles)
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("←→/Tab: Select | Enter: Continue | Esc: Back"))

	case PickerModeAddTheme:
		b.WriteString(styles.Title.Render("➕  Add Connection - Theme"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Connection: %s (%s)\n\n", m.connectionPicker.newConnName, m.connectionPicker.newConnType))
		b.WriteString("  Select a visual theme:\n\n")

		themes := ThemeNames()
		visibleCount := 8
		start := 0
		if m.connectionPicker.themeIdx >= visibleCount {
			start = m.connectionPicker.themeIdx - visibleCount + 1
		}
		end := start + visibleCount
		if end > len(themes) {
			end = len(themes)
		}

		for i := start; i < end; i++ {
			themeName := themes[i]
			theme := Themes[themeName]
			desc := theme.Description
			if i == m.connectionPicker.themeIdx {
				b.WriteString(fmt.Sprintf("  ▶ %s", styles.SelectedRow.Render(fmt.Sprintf("%-14s %s", themeName, desc))))
			} else {
				b.WriteString(fmt.Sprintf("    %-14s %s", themeName, desc))
			}
			b.WriteString("\n")
		}

		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("↑↓: Select | Enter: Save Connection | Esc: Back"))

	case PickerModeConfirmDelete:
		b.WriteString(styles.Title.Render("🗑️  Delete Connection"))
		b.WriteString("\n\n")
		if len(m.connectionPicker.connections) > 0 {
			name := m.connectionPicker.connections[m.connectionPicker.selectedIdx]
			b.WriteString(fmt.Sprintf("  Delete connection '%s'?\n\n", styles.Error.Render(name)))
			b.WriteString("  This cannot be undone.\n")
		}
		m.renderPickerError(&b, styles)
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("y: Yes, Delete | n/Esc: Cancel"))
	}

	return b.String()
}

// renderPickerError renders the error message if present
func (m Model) renderPickerError(b *strings.Builder, styles ThemedStyles) {
	if m.connectionPicker.errorMessage != "" {
		b.WriteString("\n")
		b.WriteString(styles.Error.Render("  " + m.connectionPicker.errorMessage))
		b.WriteString("\n")
	}
}
