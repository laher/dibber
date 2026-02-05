package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

// renderHighlightedQuery renders the query textarea content with SQL syntax highlighting
func (m Model) renderHighlightedQuery() string {
	tab := m.tab()
	if tab == nil {
		return ""
	}

	content := tab.textarea.Value()
	lines := strings.Split(content, "\n")

	// Get cursor position
	cursorLine := tab.textarea.Line()
	lineInfo := tab.textarea.LineInfo()
	cursorCol := lineInfo.CharOffset

	// Get textarea dimensions
	height := tab.textarea.Height()
	width := tab.textarea.Width()

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
	if tab.textarea.ShowLineNumbers {
		contentWidth = width - lineNumWidth - 1 // -1 for space after line number
	}

	// Styles for line numbers
	lineNumStyle := lipgloss.NewStyle().
		Foreground(tab.theme.TextDim).
		Width(lineNumWidth).
		Align(lipgloss.Right)

	cursorLineNumStyle := lipgloss.NewStyle().
		Foreground(tab.theme.Primary).
		Bold(true).
		Width(lineNumWidth).
		Align(lipgloss.Right)

	// Cursor style
	cursorStyle := lipgloss.NewStyle().
		Background(tab.theme.TextBright).
		Foreground(tab.theme.Secondary)

	var b strings.Builder
	isFocused := m.focus == focusQuery

	// Render visible lines
	for i := scrollOffset; i < len(lines) && i < scrollOffset+height; i++ {
		line := lines[i]

		// Line number
		if tab.textarea.ShowLineNumbers {
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

		if tab.highlighter != nil {
			highlightedLine := tab.highlighter.HighlightLine(line)

			// If this is the cursor line and we're focused, insert cursor
			if isCursorLine {
				renderedLine = m.insertCursor(tab, line, highlightedLine, cursorCol, cursorStyle, contentWidth)
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
		if tab.textarea.ShowLineNumbers {
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
func (m Model) insertCursor(tab *Tab, plainLine, highlightedLine string, cursorCol int, cursorStyle lipgloss.Style, maxWidth int) string {
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
	if beforeCursor != "" && tab.highlighter != nil {
		result.WriteString(tab.highlighter.HighlightLine(beforeCursor))
	} else if beforeCursor != "" {
		result.WriteString(beforeCursor)
	}
	result.WriteString(cursorStyle.Render(cursorChar))
	if afterCursor != "" && tab.highlighter != nil {
		result.WriteString(tab.highlighter.HighlightLine(afterCursor))
	} else if afterCursor != "" {
		result.WriteString(afterCursor)
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

	tab := m.tab()

	// Show detail view if active
	if m.focus == focusDetail && tab != nil && tab.detailView != nil {
		return m.renderDetailView()
	}

	// Show file dialog if active
	if m.focus == focusFileDialog && m.fileDialog != nil {
		return m.renderFileDialog()
	}

	// Show connection picker if active (for both existing tab switch and new tab)
	if (m.focus == focusConnectionPicker || m.focus == focusNewTabPicker) && m.connectionPicker != nil {
		return m.renderConnectionPicker()
	}

	// Get themed styles
	styles := m.GetStyles()

	// Calculate heights
	// Title: 1 line + 1 blank = 2
	// Tab bar: 1 line + 1 blank = 2
	// Query box: textarea height + 2 (border) + 1 blank = textarea.Height() + 3
	// Status bar: 1 line
	// Help: 1 line
	titleHeight := 2
	tabBarHeight := 2
	textareaHeight := 8
	if tab != nil {
		textareaHeight = tab.textarea.Height()
	}
	queryBoxHeight := textareaHeight + 4 // includes border padding and blank line
	statusHeight := 1
	helpHeight := 1
	tableHeight := m.height - titleHeight - tabBarHeight - queryBoxHeight - statusHeight - helpHeight

	if tableHeight < 3 {
		tableHeight = 3
	}

	var b strings.Builder

	// Title
	titleText := "🌱  Dibber - Database Client"
	b.WriteString(styles.Title.Render(titleText))
	b.WriteString("\n\n")

	// Tab bar
	b.WriteString(m.renderTabBar())
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

	if tab != nil && tab.result != nil {
		if tab.result.Error != nil {
			tableContent = styles.Error.Render(fmt.Sprintf("Error: %v", tab.result.Error))
		} else if len(tab.result.Rows) > 0 {
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
	if tab != nil && tab.result != nil && len(tab.result.Rows) > 0 {
		editableText := ""
		if tab.queryMeta != nil {
			if tab.queryMeta.IsEditable {
				editableText = " [Editable]"
			} else {
				editableText = " [Read-only]"
			}
		}
		statusText = fmt.Sprintf("%s%s | Page %d/%d | Row %d/%d",
			m.statusMessage, editableText, tab.currentPage+1, tab.totalPages, tab.selectedRow+1, len(tab.result.Rows))
	}
	b.WriteString(styles.StatusBar.Width(m.width).Render(statusText))
	b.WriteString("\n")

	// Help - context-sensitive
	var helpText string
	switch m.focus {
	case focusQuery:
		helpText = "Ctrl+R: Run | Ctrl+T: New Tab | Ctrl+Tab: Switch Tab | Ctrl+W: Close Tab | Ctrl+Q: Quit"
	case focusResults:
		if tab != nil && tab.result != nil && len(tab.result.Rows) > 0 {
			helpText = "↑↓: Navigate | Enter: Detail | -/+: Resize | Tab: Switch | Ctrl+Q: Quit"
		} else {
			helpText = "-/+: Resize | Tab: Switch | Ctrl+R: Run | Ctrl+Q: Quit"
		}
	default:
		helpText = "Ctrl+R: Run | Ctrl+T: New Tab | Ctrl+Tab: Switch Tab | Ctrl+P: Connections | Ctrl+Q: Quit"
	}
	b.WriteString(styles.Help.Render(helpText))

	return b.String()
}

// renderTabBar renders the tab bar showing all open tabs
func (m Model) renderTabBar() string {
	if len(m.tabs) == 0 {
		return ""
	}

	var b strings.Builder

	for i, tab := range m.tabs {
		// Determine tab label
		label := tab.connectionName
		if label == "" {
			label = tab.dbType
		}
		if label == "" {
			label = "untitled"
		}

		// Truncate long labels
		if len(label) > 15 {
			label = label[:12] + "..."
		}

		// Style based on whether this is the active tab
		var tabStyle lipgloss.Style
		if i == m.activeTab {
			tabStyle = lipgloss.NewStyle().
				Background(tab.theme.Primary).
				Foreground(tab.theme.TextBright).
				Bold(true).
				Padding(0, 1)
		} else {
			tabStyle = lipgloss.NewStyle().
				Background(tab.theme.Secondary).
				Foreground(tab.theme.TextNormal).
				Padding(0, 1)
		}

		// Add tab number for quick reference
		tabLabel := fmt.Sprintf("%d: %s", i+1, label)
		b.WriteString(tabStyle.Render(tabLabel))

		// Add separator between tabs
		if i < len(m.tabs)-1 {
			b.WriteString(" ")
		}
	}

	// Add hint for new tab
	if len(m.tabs) < 9 {
		newTabStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Padding(0, 1)
		b.WriteString(" ")
		b.WriteString(newTabStyle.Render("[+]"))
	}

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
		b.WriteString("No saved connections found. Create a encryption password to\n")
		b.WriteString("securely store your database connections.\n\n")
		b.WriteString("  encryption password (min 8 chars):\n")
		masked := strings.Repeat("•", len(m.connectionPicker.passwordInput))
		b.WriteString(fmt.Sprintf("  %s█\n", masked))
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Continue | Esc: Cancel"))

	case PickerModeConfirmVaultPassword:
		b.WriteString(styles.Title.Render("🔐  Confirm encryption password"))
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
		b.WriteString("  encryption password:\n")
		masked := strings.Repeat("•", len(m.connectionPicker.passwordInput))
		b.WriteString(fmt.Sprintf("  %s█\n", masked))
		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Unlock | Esc: Cancel"))

	case PickerModeList:
		if m.creatingNewTab {
			b.WriteString(styles.Title.Render("🔌  Select Connection for New Tab"))
		} else {
			b.WriteString(styles.Title.Render("🔌  Connection Manager"))
		}
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
				encIcon := "🔒" // encrypted
				if m.vaultManager != nil && m.vaultManager.IsPlaintextConnection(name) {
					encIcon = "📄" // plaintext
				}
				displayName := fmt.Sprintf("%s %s", encIcon, name)
				if i == m.connectionPicker.selectedIdx {
					b.WriteString(fmt.Sprintf("  ▶ %s", styles.SelectedRow.Render(displayName)))
				} else {
					b.WriteString(fmt.Sprintf("    %s", displayName))
				}
				b.WriteString("\n")
			}
		}

		m.renderPickerError(&b, styles)

		tab := m.tab()
		if tab != nil && tab.connectionName != "" {
			b.WriteString(fmt.Sprintf("\n  Current: %s", tab.connectionName))
		}

		b.WriteString("\n\n")
		if len(m.connectionPicker.connections) > 0 {
			if m.creatingNewTab {
				b.WriteString(styles.Help.Render("↑↓: Navigate | Enter: Open in new tab | Esc: Cancel"))
			} else {
				b.WriteString(styles.Help.Render("↑↓: Navigate | Enter: Connect | a: Add | d: Delete | Esc: Close"))
			}
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
		b.WriteString(styles.Help.Render("↑↓: Select | Enter: Continue | Esc: Back"))

	case PickerModeAddEncrypt:
		b.WriteString(styles.Title.Render("➕  Add Connection - Storage"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  Connection: %s (%s)\n\n", m.connectionPicker.newConnName, m.connectionPicker.newConnType))
		b.WriteString("  How should the DSN be stored?\n\n")

		options := []struct {
			name string
			desc string
		}{
			{"Encrypted", "Secure storage, requires password to unlock"},
			{"Plaintext", "No password needed (for local databases)"},
		}

		for i, opt := range options {
			if i == m.connectionPicker.encryptOptIdx {
				b.WriteString(fmt.Sprintf("  ▶ %s", styles.SelectedRow.Render(fmt.Sprintf("%-12s %s", opt.name, opt.desc))))
			} else {
				b.WriteString(fmt.Sprintf("    %-12s %s", opt.name, opt.desc))
			}
			b.WriteString("\n")
		}

		m.renderPickerError(&b, styles)
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("↑↓/Tab: Select | Enter: Save Connection | Esc: Back"))

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
