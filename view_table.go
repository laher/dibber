package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderBanner renders the startup ASCII art banner
func (m Model) renderBanner() string {
	// ASCII art for "dibber"
	banner := []string{
		`     _ _ _     _               `,
		`  __| (_) |__ | |__   ___ _ __ `,
		` / _' | | '_ \| '_ \ / _ \ '__|`,
		`| (_| | | |_) | |_) |  __/ |   `,
		` \__,_|_|_.__/|_.__/ \___|_|   `,
	}

	// Colors for each line (gradient effect)
	colors := []string{
		"#FF6B6B", // coral red
		"#FFE66D", // yellow
		"#4ECDC4", // teal
		"#45B7D1", // sky blue
		"#96CEB4", // sage green
	}

	var b strings.Builder
	b.WriteString("\n")

	for i, line := range banner {
		color := colors[i%len(colors)]
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Bold(true)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Tagline
	tagline := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Render("        A terminal database client")
	b.WriteString("\n")
	b.WriteString(tagline)
	b.WriteString("\n\n")

	// Instructions
	instructions := helpStyle.Render("  Enter a SQL query and press Ctrl+R to execute")
	b.WriteString(instructions)
	b.WriteString("\n")

	return b.String()
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
			displayLen := len(cell.String())
			if displayLen > colWidths[i] {
				colWidths[i] = displayLen
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
			displayVal := cell.String()
			cellStr := truncateString(displayVal, colWidths[i])
			cellStr = padRight(cellStr, colWidths[i])

			isSelected := actualRowIdx == m.selectedRow && m.focus == focusResults

			if cell.IsNull {
				// NULL values get special styling
				if isSelected {
					cells = append(cells, selectedRowStyle.Render(nullCellStyle.Render(cellStr)))
				} else {
					cells = append(cells, nullCellStyle.Render(cellStr))
				}
			} else if isSelected {
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
