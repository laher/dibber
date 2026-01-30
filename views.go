package main

import (
	"fmt"
	"strings"
)

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

	// Query input
	queryBoxStyle := styles.QueryBox
	if m.focus == focusQuery {
		queryBoxStyle = styles.QueryBoxFocused
	}
	b.WriteString(queryBoxStyle.Render(m.textarea.View()))
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

// renderConnectionPicker renders the connection picker dialog
func (m Model) renderConnectionPicker() string {
	styles := m.GetStyles()
	var b strings.Builder

	b.WriteString(styles.Title.Render("🔌  Switch Connection"))
	b.WriteString("\n\n")

	if m.connectionPicker.awaitPassword {
		// Password prompt mode
		b.WriteString("Enter master password to unlock saved connections:\n\n")

		// Show masked password
		masked := strings.Repeat("•", len(m.connectionPicker.passwordInput))
		b.WriteString(fmt.Sprintf("  Password: %s█\n", masked))

		if m.connectionPicker.errorMessage != "" {
			b.WriteString("\n")
			b.WriteString(styles.Error.Render("  " + m.connectionPicker.errorMessage))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(styles.Help.Render("Enter: Submit | Esc: Cancel"))
	} else {
		// Connection selection mode
		b.WriteString("Select a connection:\n\n")

		visibleCount := 10
		start := m.connectionPicker.scrollOffset
		end := start + visibleCount
		if end > len(m.connectionPicker.connections) {
			end = len(m.connectionPicker.connections)
		}

		for i := start; i < end; i++ {
			name := m.connectionPicker.connections[i]
			if i == m.connectionPicker.selectedIdx {
				// Highlight selected
				b.WriteString(fmt.Sprintf("  ▶ %s", styles.SelectedRow.Render(name)))
			} else {
				b.WriteString(fmt.Sprintf("    %s", name))
			}
			b.WriteString("\n")
		}

		if m.connectionPicker.errorMessage != "" {
			b.WriteString("\n")
			b.WriteString(styles.Error.Render("  " + m.connectionPicker.errorMessage))
			b.WriteString("\n")
		}

		// Show current connection if any
		if m.connectionName != "" {
			b.WriteString(fmt.Sprintf("\n  Current: %s", m.connectionName))
		}

		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("↑↓: Navigate | Enter: Connect | Esc: Cancel"))
	}

	return b.String()
}
