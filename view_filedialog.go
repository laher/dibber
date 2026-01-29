package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFileDialog renders the file open dialog
func (m Model) renderFileDialog() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🌱  Dibber - Database Client"))
	b.WriteString("\n\n")

	// Dialog title
	dialogTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFE66D")).
		Render("📂 Open SQL File")
	b.WriteString(dialogTitle)
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("   Directory: %s", m.fileDialog.directory)))
	b.WriteString("\n\n")

	// Entry list
	visibleCount := 10
	if m.height > 20 {
		visibleCount = m.height - 15
	}

	endIdx := m.fileDialog.scrollOffset + visibleCount
	if endIdx > len(m.fileDialog.entries) {
		endIdx = len(m.fileDialog.entries)
	}

	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AFAFAF"))
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ECDC4"))
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)

	for i := m.fileDialog.scrollOffset; i < endIdx; i++ {
		entry := m.fileDialog.entries[i]
		var displayName string
		var icon string

		if entry.isDir {
			if entry.name == ".." {
				icon = "📁 "
				displayName = ".. (parent directory)"
			} else {
				icon = "📁 "
				displayName = entry.name + "/"
			}
		} else {
			icon = "📄 "
			displayName = entry.name
		}

		if i == m.fileDialog.selectedIdx {
			b.WriteString(selectedStyle.Render("▶ " + icon + displayName))
		} else {
			prefix := "  " + icon
			if entry.isDir {
				b.WriteString(dirStyle.Render(prefix + displayName))
			} else {
				b.WriteString(fileStyle.Render(prefix + displayName))
			}
		}
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.fileDialog.entries) > visibleCount {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("   Showing %d-%d of %d items",
			m.fileDialog.scrollOffset+1, endIdx, len(m.fileDialog.entries))))
		b.WriteString("\n")
	}

	// Pad to fill screen
	linesUsed := 6 + (endIdx - m.fileDialog.scrollOffset)
	if len(m.fileDialog.entries) > visibleCount {
		linesUsed += 2
	}
	for i := linesUsed; i < m.height-3; i++ {
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(statusBarStyle.Width(m.width).Render(m.statusMessage))
	b.WriteString("\n")

	// Help
	b.WriteString(helpStyle.Render("↑↓: Select | Enter: Open/Navigate | Esc: Cancel"))

	return b.String()
}
