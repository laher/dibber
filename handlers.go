package main

import (
	"errors"
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

	case "ctrl+n":
		// Toggle NULL state for focused field
		if m.queryMeta != nil && m.queryMeta.IsEditable {
			idx := m.detailView.focusedField
			m.detailView.isNull[idx] = !m.detailView.isNull[idx]
			if m.detailView.isNull[idx] {
				// Clear the input when setting to NULL
				m.detailView.inputs[idx].SetValue("")
				m.statusMessage = "Field set to NULL"
			} else {
				m.statusMessage = "Field set to non-NULL (empty string)"
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
		origVal := m.detailView.originalValues[m.detailView.focusedField]
		if !origVal.IsNull && strings.Contains(origVal.Value, "\n") {
			lines := strings.Split(origVal.Value, "\n")
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
			idx := m.detailView.focusedField
			var cmd tea.Cmd
			m.detailView.inputs[idx], cmd = m.detailView.inputs[idx].Update(msg)
			cmds = append(cmds, cmd)

			// If user types in a NULL field, automatically make it non-NULL
			if m.detailView.isNull[idx] && m.detailView.inputs[idx].Value() != "" {
				m.detailView.isNull[idx] = false
			}
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

// handleConnectionPickerKeys handles key events in the connection picker/manager
func (m Model) handleConnectionPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.connectionPicker == nil {
		return m, nil
	}

	switch m.connectionPicker.mode {
	case PickerModeCreateVault:
		return m.handleCreateVaultMode(msg)
	case PickerModeConfirmVaultPassword:
		return m.handleConfirmVaultPasswordMode(msg)
	case PickerModeUnlock:
		return m.handleUnlockMode(msg)
	case PickerModeList:
		return m.handleListMode(msg)
	case PickerModeAddName:
		return m.handleAddNameMode(msg)
	case PickerModeAddDSN:
		return m.handleAddDSNMode(msg)
	case PickerModeAddType:
		return m.handleAddTypeMode(msg)
	case PickerModeAddTheme:
		return m.handleAddThemeMode(msg)
	case PickerModeConfirmDelete:
		return m.handleConfirmDeleteMode(msg)
	}
	return m, nil
}

// handleCreateVaultMode handles creating a new vault (first-time setup)
func (m Model) handleCreateVaultMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.closeConnectionPicker("Cancelled")
	case "enter":
		if len(m.connectionPicker.passwordInput) < 8 {
			m.connectionPicker.errorMessage = "Password must be at least 8 characters"
			return m, nil
		}
		m.connectionPicker.mode = PickerModeConfirmVaultPassword
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "backspace":
		if len(m.connectionPicker.passwordInput) > 0 {
			m.connectionPicker.passwordInput = m.connectionPicker.passwordInput[:len(m.connectionPicker.passwordInput)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.connectionPicker.passwordInput += msg.String()
		}
		return m, nil
	}
}

// handleConfirmVaultPasswordMode handles confirming the vault password
func (m Model) handleConfirmVaultPasswordMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.connectionPicker.mode = PickerModeCreateVault
		m.connectionPicker.confirmPasswordInput = ""
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "enter":
		if m.connectionPicker.confirmPasswordInput != m.connectionPicker.passwordInput {
			m.connectionPicker.errorMessage = "Passwords do not match"
			m.connectionPicker.confirmPasswordInput = ""
			return m, nil
		}
		// Create the vault
		if err := m.vaultManager.InitializeWithPassword(m.connectionPicker.passwordInput); err != nil {
			m.connectionPicker.errorMessage = "Failed to create vault: " + err.Error()
			return m, nil
		}
		// Move to add first connection
		m.connectionPicker.mode = PickerModeAddName
		m.connectionPicker.passwordInput = ""
		m.connectionPicker.confirmPasswordInput = ""
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "backspace":
		if len(m.connectionPicker.confirmPasswordInput) > 0 {
			m.connectionPicker.confirmPasswordInput = m.connectionPicker.confirmPasswordInput[:len(m.connectionPicker.confirmPasswordInput)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.connectionPicker.confirmPasswordInput += msg.String()
		}
		return m, nil
	}
}

// handleUnlockMode handles unlocking an existing vault
func (m Model) handleUnlockMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.closeConnectionPicker("Cancelled")
	case "enter":
		if m.connectionPicker.passwordInput == "" {
			m.connectionPicker.errorMessage = "Password required"
			return m, nil
		}
		if err := m.vaultManager.Unlock(m.connectionPicker.passwordInput); err != nil {
			if errors.Is(err, ErrDecryptionFailed) {
				m.connectionPicker.errorMessage = "Incorrect password"
			} else {
				m.connectionPicker.errorMessage = err.Error()
			}
			m.connectionPicker.passwordInput = ""
			return m, nil
		}
		// Unlocked - refresh connections and go to list
		m.connectionPicker.connections = m.vaultManager.ListConnections()
		m.connectionPicker.mode = PickerModeList
		m.connectionPicker.passwordInput = ""
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "backspace":
		if len(m.connectionPicker.passwordInput) > 0 {
			m.connectionPicker.passwordInput = m.connectionPicker.passwordInput[:len(m.connectionPicker.passwordInput)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.connectionPicker.passwordInput += msg.String()
		}
		return m, nil
	}
}

// handleListMode handles the connection list view
func (m Model) handleListMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.closeConnectionPicker("Closed")
	case "enter":
		// Switch to selected connection
		if len(m.connectionPicker.connections) > 0 {
			selectedName := m.connectionPicker.connections[m.connectionPicker.selectedIdx]
			if err := m.switchConnection(selectedName); err != nil {
				m.connectionPicker.errorMessage = err.Error()
				return m, nil
			}
			m.focus = focusQuery
			m.connectionPicker = nil
			m.statusMessage = "Switched to: " + selectedName
			m.textarea.Focus()
		}
		return m, nil
	case "a", "n":
		// Add new connection
		m.connectionPicker.mode = PickerModeAddName
		m.connectionPicker.newConnName = ""
		m.connectionPicker.newConnDSN = ""
		m.connectionPicker.newConnType = ""
		m.connectionPicker.newConnTheme = ""
		m.connectionPicker.themeIdx = 0
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "d", "x":
		// Delete selected connection
		if len(m.connectionPicker.connections) > 0 {
			m.connectionPicker.mode = PickerModeConfirmDelete
			m.connectionPicker.errorMessage = ""
		}
		return m, nil
	case "up", "k":
		if m.connectionPicker.selectedIdx > 0 {
			m.connectionPicker.selectedIdx--
			if m.connectionPicker.selectedIdx < m.connectionPicker.scrollOffset {
				m.connectionPicker.scrollOffset = m.connectionPicker.selectedIdx
			}
		}
		return m, nil
	case "down", "j":
		if m.connectionPicker.selectedIdx < len(m.connectionPicker.connections)-1 {
			m.connectionPicker.selectedIdx++
			visibleCount := 10
			if m.connectionPicker.selectedIdx >= m.connectionPicker.scrollOffset+visibleCount {
				m.connectionPicker.scrollOffset = m.connectionPicker.selectedIdx - visibleCount + 1
			}
		}
		return m, nil
	}
	return m, nil
}

// handleAddNameMode handles entering the connection name
func (m Model) handleAddNameMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.connectionPicker.mode = PickerModeList
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.connectionPicker.newConnName)
		if name == "" {
			m.connectionPicker.errorMessage = "Name is required"
			return m, nil
		}
		// Check for duplicate
		for _, existing := range m.connectionPicker.connections {
			if existing == name {
				m.connectionPicker.errorMessage = "Connection '" + name + "' already exists"
				return m, nil
			}
		}
		m.connectionPicker.newConnName = name
		m.connectionPicker.mode = PickerModeAddDSN
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "backspace":
		if len(m.connectionPicker.newConnName) > 0 {
			m.connectionPicker.newConnName = m.connectionPicker.newConnName[:len(m.connectionPicker.newConnName)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.connectionPicker.newConnName += msg.String()
		}
		return m, nil
	}
}

// handleAddDSNMode handles entering the DSN (masked)
func (m Model) handleAddDSNMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.connectionPicker.mode = PickerModeAddName
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "enter":
		dsn := strings.TrimSpace(m.connectionPicker.newConnDSN)
		if dsn == "" {
			m.connectionPicker.errorMessage = "DSN is required"
			return m, nil
		}
		m.connectionPicker.newConnDSN = dsn
		// Auto-detect type
		m.connectionPicker.newConnType = detectDBType(dsn)
		m.connectionPicker.mode = PickerModeAddType
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "backspace":
		if len(m.connectionPicker.newConnDSN) > 0 {
			m.connectionPicker.newConnDSN = m.connectionPicker.newConnDSN[:len(m.connectionPicker.newConnDSN)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.connectionPicker.newConnDSN += msg.String()
		}
		return m, nil
	}
}

// handleAddTypeMode handles selecting the database type
func (m Model) handleAddTypeMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	types := []string{"mysql", "postgres", "sqlite"}
	currentIdx := 0
	for i, t := range types {
		if t == m.connectionPicker.newConnType {
			currentIdx = i
			break
		}
	}

	switch msg.String() {
	case "esc":
		m.connectionPicker.mode = PickerModeAddDSN
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "enter":
		m.connectionPicker.mode = PickerModeAddTheme
		m.connectionPicker.themeIdx = 0
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "left", "h":
		if currentIdx > 0 {
			m.connectionPicker.newConnType = types[currentIdx-1]
		}
		return m, nil
	case "right", "l":
		if currentIdx < len(types)-1 {
			m.connectionPicker.newConnType = types[currentIdx+1]
		}
		return m, nil
	case "tab":
		m.connectionPicker.newConnType = types[(currentIdx+1)%len(types)]
		return m, nil
	}
	return m, nil
}

// handleAddThemeMode handles selecting the theme
func (m Model) handleAddThemeMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	themes := ThemeNames()

	switch msg.String() {
	case "esc":
		m.connectionPicker.mode = PickerModeAddType
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "enter":
		// Save the connection
		theme := themes[m.connectionPicker.themeIdx]
		if theme == "default" {
			theme = ""
		}
		err := m.vaultManager.AddConnection(
			m.connectionPicker.newConnName,
			m.connectionPicker.newConnDSN,
			m.connectionPicker.newConnType,
			theme,
		)
		if err != nil {
			m.connectionPicker.errorMessage = "Failed to save: " + err.Error()
			return m, nil
		}
		// Refresh and go back to list
		m.connectionPicker.connections = m.vaultManager.ListConnections()
		m.connectionPicker.mode = PickerModeList
		m.connectionPicker.errorMessage = ""
		m.connectionPicker.newConnName = ""
		m.connectionPicker.newConnDSN = ""
		// Select the new connection
		for i, name := range m.connectionPicker.connections {
			if name == m.connectionPicker.newConnName {
				m.connectionPicker.selectedIdx = i
				break
			}
		}
		return m, nil
	case "up", "k":
		if m.connectionPicker.themeIdx > 0 {
			m.connectionPicker.themeIdx--
		}
		return m, nil
	case "down", "j":
		if m.connectionPicker.themeIdx < len(themes)-1 {
			m.connectionPicker.themeIdx++
		}
		return m, nil
	}
	return m, nil
}

// handleConfirmDeleteMode handles confirming connection deletion
func (m Model) handleConfirmDeleteMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.connectionPicker.mode = PickerModeList
		m.connectionPicker.errorMessage = ""
		return m, nil
	case "y":
		if len(m.connectionPicker.connections) == 0 {
			return m, nil
		}
		name := m.connectionPicker.connections[m.connectionPicker.selectedIdx]
		if err := m.vaultManager.RemoveConnection(name); err != nil {
			m.connectionPicker.errorMessage = "Failed to delete: " + err.Error()
			m.connectionPicker.mode = PickerModeList
			return m, nil
		}
		// Refresh list
		m.connectionPicker.connections = m.vaultManager.ListConnections()
		if m.connectionPicker.selectedIdx >= len(m.connectionPicker.connections) {
			m.connectionPicker.selectedIdx = len(m.connectionPicker.connections) - 1
			if m.connectionPicker.selectedIdx < 0 {
				m.connectionPicker.selectedIdx = 0
			}
		}
		m.connectionPicker.mode = PickerModeList
		m.connectionPicker.errorMessage = ""
		return m, nil
	}
	return m, nil
}

// closeConnectionPicker closes the picker and returns to query view
func (m Model) closeConnectionPicker(message string) (tea.Model, tea.Cmd) {
	m.focus = focusQuery
	m.connectionPicker = nil
	m.statusMessage = message
	m.textarea.Focus()
	return m, nil
}
