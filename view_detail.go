package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	b.WriteString(titleStyle.Render("🌱  Dibber - Database Client"))
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
