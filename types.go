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

// ColumnType represents the general category of a database column type
type ColumnType string

const (
	ColTypeText     ColumnType = "text"
	ColTypeNumeric  ColumnType = "numeric"
	ColTypeBoolean  ColumnType = "boolean"
	ColTypeDatetime ColumnType = "datetime"
	ColTypeBlob     ColumnType = "blob"
	ColTypeUnknown  ColumnType = "unknown"
)

// CellValue represents a database cell value with NULL awareness
type CellValue struct {
	Value  string
	IsNull bool
}

// String returns a display string for the cell value
func (c CellValue) String() string {
	if c.IsNull {
		return "<NULL>"
	}
	return c.Value
}

// QueryResult holds the result of a SQL query
type QueryResult struct {
	Columns     []string
	ColumnTypes []ColumnType
	Rows        [][]CellValue
	Error       error
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
	originalValues      []CellValue // original cell values with NULL info
	inputs              []textinput.Model
	isNull              []bool       // current NULL state per field (can be toggled)
	columnTypes         []ColumnType // type info for SQL generation
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

// IsNumeric returns true if the column type is numeric
func (ct ColumnType) IsNumeric() bool {
	return ct == ColTypeNumeric
}

// IsBoolean returns true if the column type is boolean
func (ct ColumnType) IsBoolean() bool {
	return ct == ColTypeBoolean
}

// IsText returns true if the column type is text-like
func (ct ColumnType) IsText() bool {
	return ct == ColTypeText || ct == ColTypeDatetime || ct == ColTypeUnknown
}
