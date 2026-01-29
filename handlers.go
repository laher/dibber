package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// handleFileDialogKeys handles key events in the file dialog
func (m Model) handleFileDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focus = focusQuery
		m.fileDialog = nil
		m.statusMessage = "Open cancelled"
		return m, nil
	case "enter":
		if len(m.fileDialog.entries) > 0 {
			selected := m.fileDialog.entries[m.fileDialog.selectedIdx]
			m.navigateFileDialog(selected)
		}
		return m, nil
	case "up", "k":
		if m.fileDialog.selectedIdx > 0 {
			m.fileDialog.selectedIdx--
			if m.fileDialog.selectedIdx < m.fileDialog.scrollOffset {
				m.fileDialog.scrollOffset = m.fileDialog.selectedIdx
			}
		}
		return m, nil
	case "down", "j":
		if m.fileDialog.selectedIdx < len(m.fileDialog.entries)-1 {
			m.fileDialog.selectedIdx++
			visibleCount := 10
			if m.fileDialog.selectedIdx >= m.fileDialog.scrollOffset+visibleCount {
				m.fileDialog.scrollOffset = m.fileDialog.selectedIdx - visibleCount + 1
			}
		}
		return m, nil
	default:
		return m, nil
	}
}

// handleDetailViewKeys handles key events in the detail view
func (m Model) handleDetailViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		// Close detail view, go back to results
		m.focus = focusResults
		m.detailView = nil
		return m, nil

	case "f5", "ctrl+u":
		// Generate UPDATE and append to query window
		if m.queryMeta != nil && m.queryMeta.IsEditable {
			updateSQL := m.generateUpdateSQL()
			if updateSQL != "" {
				m.appendQueryToTextarea(updateSQL)
				m.focus = focusQuery
				m.textarea.Focus()
				m.detailView = nil
				m.statusMessage = "UPDATE statement appended. Press Ctrl+R to execute."
				return m, nil
			}
			m.statusMessage = "No changes to update."
		}
		return m, nil

	case "f6", "ctrl+d":
		// Generate DELETE and append to query window
		if m.queryMeta != nil && m.queryMeta.IsEditable {
			deleteSQL := m.generateDeleteSQL()
			if deleteSQL != "" {
				m.appendQueryToTextarea(deleteSQL)
				m.focus = focusQuery
				m.textarea.Focus()
				m.detailView = nil
				m.statusMessage = "DELETE statement appended. Press Ctrl+R to execute."
				return m, nil
			}
		}
		return m, nil

	case "f7", "ctrl+i":
		// Generate INSERT and append to query window
		if m.queryMeta != nil && m.queryMeta.IsEditable {
			insertSQL := m.generateInsertSQL()
			if insertSQL != "" {
				m.appendQueryToTextarea(insertSQL)
				m.focus = focusQuery
				m.textarea.Focus()
				m.detailView = nil
				m.statusMessage = "INSERT statement appended. Press Ctrl+R to execute."
				return m, nil
			}
		}
		return m, nil

	case "up", "shift+tab":
		if m.detailView.focusedField > 0 {
			m.detailView.inputs[m.detailView.focusedField].Blur()
			m.detailView.focusedField--
			m.detailView.contentScrollOffset = 0 // Reset content scroll when changing fields
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
			m.detailView.contentScrollOffset = 0 // Reset content scroll when changing fields
			m.detailView.inputs[m.detailView.focusedField].Focus()
			// Adjust scroll if needed
			if m.detailView.focusedField >= m.detailView.scrollOffset+m.detailView.visibleFields {
				m.detailView.scrollOffset = m.detailView.focusedField - m.detailView.visibleFields + 1
			}
		}
		return m, nil

	case "pgdown":
		// Scroll down within multi-line content
		val := m.detailView.originalRow[m.detailView.focusedField]
		if strings.Contains(val, "\n") {
			lines := strings.Split(val, "\n")
			maxScroll := len(lines) - 10 // Keep at least 10 lines visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.detailView.contentScrollOffset += 10
			if m.detailView.contentScrollOffset > maxScroll {
				m.detailView.contentScrollOffset = maxScroll
			}
		}
		return m, nil

	case "pgup":
		// Scroll up within multi-line content
		m.detailView.contentScrollOffset -= 10
		if m.detailView.contentScrollOffset < 0 {
			m.detailView.contentScrollOffset = 0
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

// handleResultsNavigation handles navigation keys in the results view
func (m Model) handleResultsNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	return m, nil
}
