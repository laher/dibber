package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderDetailView renders the detail/edit view for a row
func (m Model) renderDetailView() string {
	styles := m.GetStyles()

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
	titleText := "🌱  Dibber - Database Client"
	if m.connectionName != "" {
		titleText = fmt.Sprintf("🌱  Dibber - %s (%s)", m.connectionName, m.dbType)
	}
	b.WriteString(styles.Title.Render(titleText))
	b.WriteString("\n\n")

	// Detail view header
	editableStatus := ""
	if m.queryMeta != nil && m.queryMeta.IsEditable {
		editableStatus = styles.EditableBadge.Render(" [EDITABLE]")
	} else {
		editableStatus = styles.ReadOnlyBadge.Render(" [READ-ONLY]")
	}
	b.WriteString(styles.DetailTitle.Render(fmt.Sprintf("Row Detail - Row %d%s", m.detailView.rowIndex+1, editableStatus)))
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
		colType := m.detailView.columnTypes[i]
		isNull := m.detailView.isNull[i]
		isFocused := i == m.detailView.focusedField

		// Build label with type indicator and NULL badge
		labelText := colName + ":"
		label := styles.FieldLabel.Render(labelText)

		// Add NULL badge if field is NULL
		nullBadge := ""
		if isNull {
			nullBadge = styles.NullBadge.Render(" [NULL]")
		}

		// Add type indicator for editable fields
		typeIndicator := ""
		if m.queryMeta != nil && m.queryMeta.IsEditable {
			switch colType {
			case ColTypeNumeric:
				typeIndicator = styles.Help.Render(" #")
			case ColTypeBoolean:
				typeIndicator = styles.Help.Render(" ✓")
			}
		}

		if m.queryMeta != nil && m.queryMeta.IsEditable {
			// Editable field
			if isNull {
				// Show NULL placeholder instead of input
				nullDisplay := styles.NullValue.Render("<NULL>")
				if isFocused {
					nullDisplay = lipgloss.NewStyle().
						Foreground(m.theme.SyntaxNull).
						Background(m.theme.Secondary).
						Bold(true).
						Render("<NULL>")
				}
				b.WriteString(fmt.Sprintf("%s%s%s %s\n", label, typeIndicator, nullBadge, nullDisplay))
			} else {
				inputView := m.detailView.inputs[i].View()
				inputVal := m.detailView.inputs[i].Value()

				// Show empty string indicator
				if inputVal == "" {
					emptyIndicator := styles.EmptyString.Render(`""`)
					if isFocused {
						b.WriteString(fmt.Sprintf("%s%s %s %s\n", label, typeIndicator, inputView, emptyIndicator))
					} else {
						b.WriteString(fmt.Sprintf("%s%s %s %s\n", label, typeIndicator, styles.FieldInput.Render(inputView), emptyIndicator))
					}
				} else {
					// Regular value with type-aware styling
					if isFocused {
						b.WriteString(fmt.Sprintf("%s%s %s\n", label, typeIndicator, inputView))
					} else {
						b.WriteString(fmt.Sprintf("%s%s %s\n", label, typeIndicator, styles.FieldInput.Render(inputView)))
					}
				}
			}
			linesWritten++
		} else {
			// Read-only field - handle multi-line content
			origVal := m.detailView.originalValues[i]

			if origVal.IsNull {
				// NULL value
				nullDisplay := styles.NullValue.Render("<NULL>")
				if isFocused {
					nullDisplay = lipgloss.NewStyle().
						Foreground(m.theme.SyntaxNull).
						Background(m.theme.Secondary).
						Bold(true).
						Render("<NULL>")
				}
				b.WriteString(fmt.Sprintf("%s%s %s\n", label, nullBadge, nullDisplay))
				linesWritten++
			} else if strings.Contains(origVal.Value, "\n") {
				// Multi-line value - display as a block
				b.WriteString(label)
				b.WriteString("\n")
				linesWritten++

				lines := strings.Split(origVal.Value, "\n")
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
					Foreground(m.theme.SyntaxString).
					PaddingLeft(2)

				if isFocused {
					blockStyle = blockStyle.
						Background(m.theme.Secondary).
						Foreground(m.theme.Warning)
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
						b.WriteString(styles.Help.Render(fmt.Sprintf("  ↑ %d lines above | ↓ %d lines below (PgUp/PgDn to scroll)", scrollOffset, remaining)))
					} else if scrollOffset > 0 {
						b.WriteString(styles.Help.Render(fmt.Sprintf("  ↑ %d lines above (PgUp to scroll)", scrollOffset)))
					} else {
						b.WriteString(styles.Help.Render(fmt.Sprintf("  ↓ %d more lines (PgDn to scroll)", remaining)))
					}
					b.WriteString("\n")
					linesWritten++
				} else if endLine < totalLines {
					b.WriteString(styles.Help.Render(fmt.Sprintf("  ... (%d more lines)", totalLines-endLine)))
					b.WriteString("\n")
					linesWritten++
				}
			} else {
				// Single-line value
				val := origVal.Value
				displayVal := val

				// Show empty string indicator
				if val == "" {
					displayVal = `""`
					style := styles.EmptyString
					if isFocused {
						style = lipgloss.NewStyle().
							Foreground(m.theme.TextDim).
							Background(m.theme.Secondary).
							Italic(true)
					}
					b.WriteString(fmt.Sprintf("%s %s\n", label, style.Render(displayVal)))
				} else {
					// Truncate if too long, show more for focused
					maxLen := 60
					if isFocused {
						maxLen = m.width - 25
					}
					if len(displayVal) > maxLen {
						displayVal = displayVal[:maxLen-3] + "..."
					}

					// Type-aware styling
					var style lipgloss.Style
					switch colType {
					case ColTypeNumeric:
						style = styles.NumericValue
					case ColTypeBoolean:
						style = styles.BooleanValue
					default:
						style = styles.FieldValue
					}

					if isFocused {
						style = style.Background(m.theme.Secondary)
					}
					b.WriteString(fmt.Sprintf("%s %s\n", label, style.Render(displayVal)))
				}
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
	b.WriteString(styles.StatusBar.Width(m.width).Render(m.statusMessage))
	b.WriteString("\n")

	// Help
	var helpText string
	if m.queryMeta != nil && m.queryMeta.IsEditable {
		helpText = "↑↓: Navigate | Ctrl+N: Toggle NULL | Ctrl+U/D/I: UPDATE/DELETE/INSERT | Esc: Back"
	} else {
		helpText = "↑↓/Tab: Navigate fields | PgUp/PgDn: Scroll content | Esc: Back | Ctrl+Q: Quit"
	}
	b.WriteString(styles.Help.Render(helpText))

	return b.String()
}
