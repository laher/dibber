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
	b.WriteString(titleStyle.Render("🌱  Dibber - Database Client"))
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
		tableContent = m.renderBanner()
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
	helpText := "Ctrl+R: Run | Ctrl+S: Save | Ctrl+O: Open | Tab: Focus | Enter: Detail | -/+: Resize | Ctrl+Q: Quit"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}
