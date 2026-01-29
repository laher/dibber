package main

import "github.com/charmbracelet/bubbles/textinput"

// Focus states
type focusState int

const (
	focusQuery focusState = iota
	focusResults
	focusDetail
	focusFileDialog
)

// QueryResult holds the result of a SQL query
type QueryResult struct {
	Columns []string
	Rows    [][]string
	Error   error
}

// QueryMeta holds parsed metadata about the query
type QueryMeta struct {
	TableName  string
	IsEditable bool
	IDColumn   string
	IDIndex    int
}

// DetailView holds the state for the detail/edit view
type DetailView struct {
	rowIndex            int
	originalRow         []string
	inputs              []textinput.Model
	focusedField        int
	scrollOffset        int
	visibleFields       int
	contentScrollOffset int // scroll offset within a multi-line field
}

// FileDialogEntry represents a file or directory in the file dialog
type FileDialogEntry struct {
	name  string
	isDir bool
}

// FileDialog holds the state for the file open dialog
type FileDialog struct {
	entries      []FileDialogEntry
	selectedIdx  int
	scrollOffset int
	directory    string
}
