package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// openFileDialog opens the file selection dialog
func (m *Model) openFileDialog() {
	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error getting directory: %v", err)
		return
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
	data, err := os.ReadFile(filename)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Error reading file: %v", err)
		return
	}

	content := string(data)
	m.textarea.SetValue(content)
	m.sqlFile = filename
	m.lastSavedContent = content
	m.statusMessage = fmt.Sprintf("Opened %s", filename)

	// Clear any existing results
	m.result = nil
	m.queryMeta = nil
}

// saveToFile saves the current textarea content to the SQL file
func (m *Model) saveToFile() {
	if m.sqlFile == "" {
		return
	}
	content := m.textarea.Value()
	// Write file, ignoring errors (we don't want to crash on save failure)
	if err := os.WriteFile(m.sqlFile, []byte(content), 0644); err == nil {
		m.lastSavedContent = content
	}
}

// hasUnsavedChanges returns true if the textarea content differs from the last saved content
func (m Model) hasUnsavedChanges() bool {
	return m.textarea.Value() != m.lastSavedContent
}

// appendQueryToTextarea appends a SQL statement to the textarea and moves cursor to end
func (m *Model) appendQueryToTextarea(sql string) {
	current := m.textarea.Value()
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

	m.textarea.SetValue(newContent)
	// Move cursor to end
	m.textarea.CursorEnd()
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
