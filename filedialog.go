package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// openFileDialog opens the file selection dialog
func (m *Model) openFileDialog() {
	tab := m.activeTabPtr()
	// Use the configured SQL directory
	dir := m.sqlDir
	if tab != nil && tab.sqlDir != "" {
		dir = tab.sqlDir
	}
	if dir == "" {
		// Fallback to current directory if sqlDir not set
		var err error
		dir, err = os.Getwd()
		if err != nil {
			m.statusMessage = fmt.Sprintf("Error getting directory: %v", err)
			return
		}
	}

	m.loadDirectoryIntoDialog(dir)
}

// loadDirectoryIntoDialog loads the contents of a directory into the file dialog
func (m *Model) loadDirectoryIntoDialog(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error reading directory: %v", err)
		return
	}

	var dialogEntries []FileDialogEntry

	// Add parent directory entry (if not at root)
	if dir != "/" {
		dialogEntries = append(dialogEntries, FileDialogEntry{name: "..", isDir: true})
	}

	// Add directories first
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dialogEntries = append(dialogEntries, FileDialogEntry{name: entry.Name(), isDir: true})
		}
	}

	// Then add .sql files
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			dialogEntries = append(dialogEntries, FileDialogEntry{name: entry.Name(), isDir: false})
		}
	}

	if len(dialogEntries) == 0 {
		m.statusMessage = "No .sql files or directories found"
		return
	}

	m.fileDialog = &FileDialog{
		entries:     dialogEntries,
		selectedIdx: 0,
		directory:   dir,
	}
	m.focus = focusFileDialog
	m.statusMessage = "Select a file or directory"
}

// loadFile loads the selected SQL file into the textarea
func (m *Model) loadFile(filename string) {
	tab := m.activeTabPtr()
	if tab == nil {
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error reading file: %v", err)
		return
	}

	content := string(data)
	tab.textarea.SetValue(content)
	tab.sqlFile = filename
	tab.lastSavedContent = content
	m.statusMessage = fmt.Sprintf("Opened %s", filename)

	// Clear any existing results
	tab.result = nil
	tab.queryMeta = nil
}

// saveToFile saves the current textarea content to the SQL file
func (m *Model) saveToFile() {
	tab := m.activeTabPtr()
	if tab == nil || tab.sqlFile == "" {
		return
	}
	content := tab.textarea.Value()
	// Write file, ignoring errors (we don't want to crash on save failure)
	if err := os.WriteFile(tab.sqlFile, []byte(content), 0644); err == nil {
		tab.lastSavedContent = content
	}
}

// reloadFileFromDisk reloads the SQL file from disk into the textarea
func (m *Model) reloadFileFromDisk() {
	tab := m.activeTabPtr()
	if tab == nil || tab.sqlFile == "" {
		return
	}

	data, err := os.ReadFile(tab.sqlFile)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error reloading file: %v", err)
		return
	}

	content := string(data)
	tab.textarea.SetValue(content)
	tab.lastSavedContent = content
	m.statusMessage = fmt.Sprintf("Reloaded %s", tab.sqlFile)
}

// hasUnsavedChanges returns true if the active tab's textarea content differs from the last saved content
func (m Model) hasUnsavedChanges() bool {
	tab := m.tab()
	if tab == nil {
		return false
	}
	return tab.textarea.Value() != tab.lastSavedContent
}

// appendQueryToTextarea appends a SQL statement to the textarea and moves cursor to end
func (m *Model) appendQueryToTextarea(sql string) {
	tab := m.activeTabPtr()
	if tab == nil {
		return
	}

	current := tab.textarea.Value()
	var newContent string

	if strings.TrimSpace(current) == "" {
		newContent = sql + ";"
	} else {
		// Ensure current content ends properly
		current = strings.TrimRight(current, " \t\n")
		if !strings.HasSuffix(current, ";") {
			current += ";"
		}
		newContent = current + "\n\n" + sql + ";"
	}

	tab.textarea.SetValue(newContent)

	// Navigate to the last line, then to the end of that line
	// This ensures the textarea scrolls to show the new content
	totalLines := tab.textarea.LineCount()
	for i := 0; i < totalLines; i++ {
		tab.textarea.CursorDown()
	}
	tab.textarea.CursorEnd()

	// Save to file
	m.saveToFile()
}

// navigateFileDialog handles navigation within the file dialog
func (m *Model) navigateFileDialog(selected FileDialogEntry) {
	if selected.isDir {
		// Navigate into directory
		var newDir string
		if selected.name == ".." {
			newDir = filepath.Dir(m.fileDialog.directory)
		} else {
			newDir = filepath.Join(m.fileDialog.directory, selected.name)
		}
		m.loadDirectoryIntoDialog(newDir)
	} else {
		// Open the file
		fullPath := filepath.Join(m.fileDialog.directory, selected.name)
		m.loadFile(fullPath)
		m.focus = focusQuery
		m.fileDialog = nil
	}
}
