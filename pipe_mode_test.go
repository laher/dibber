package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestOutputTable tests table output formatting
func TestOutputTable(t *testing.T) {
	columns := []string{"id", "name", "status"}
	rows := [][]string{
		{"1", "Alice", "active"},
		{"2", "Bob", "inactive"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = wErr

	outputTable(columns, rows)

	_ = w.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	var bufErr bytes.Buffer
	_, _ = io.Copy(&bufErr, rErr)

	// Check output contains expected elements
	if !strings.Contains(output, "id") {
		t.Error("Output should contain 'id' column")
	}
	if !strings.Contains(output, "Alice") {
		t.Error("Output should contain 'Alice'")
	}
	if !strings.Contains(output, "---") {
		t.Error("Output should contain separator line")
	}
}

// TestOutputCSV tests CSV output formatting
func TestOutputCSV(t *testing.T) {
	columns := []string{"id", "name", "note"}
	rows := [][]string{
		{"1", "Alice", "simple"},
		{"2", "Bob", "has, comma"},
		{"3", "Charlie", `has "quotes"`},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCSV(columns, rows, ",")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Check header
	if lines[0] != "id,name,note" {
		t.Errorf("Header = %q, want %q", lines[0], "id,name,note")
	}

	// Check simple row
	if lines[1] != "1,Alice,simple" {
		t.Errorf("Row 1 = %q, want %q", lines[1], "1,Alice,simple")
	}

	// Check row with comma (should be quoted)
	if !strings.Contains(lines[2], `"has, comma"`) {
		t.Errorf("Row 2 should have quoted comma field, got %q", lines[2])
	}

	// Check row with quotes (should be escaped)
	if !strings.Contains(lines[3], `"has ""quotes"""`) {
		t.Errorf("Row 3 should have escaped quotes, got %q", lines[3])
	}
}

// TestOutputTSV tests TSV output formatting
func TestOutputTSV(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCSV(columns, rows, "\t")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Check header uses tabs
	if lines[0] != "id\tname" {
		t.Errorf("Header = %q, want %q", lines[0], "id\tname")
	}

	// Check data row uses tabs
	if lines[1] != "1\tAlice" {
		t.Errorf("Row 1 = %q, want %q", lines[1], "1\tAlice")
	}
}

// TestPadAndTruncate tests string padding and truncation
func TestPadAndTruncate(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"abc", 5, "abc  "},
		{"abcdef", 4, "a..."},
		{"ab", 2, "ab"},
		{"hello\nworld", 10, "hello...  "},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := padAndTruncate(tc.input, tc.width)
			if result != tc.expected {
				t.Errorf("padAndTruncate(%q, %d) = %q, want %q",
					tc.input, tc.width, result, tc.expected)
			}
		})
	}
}

// TestOutputTableEmpty tests table output with no rows
func TestOutputTableEmpty(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{}

	// Capture stdout
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = wErr

	outputTable(columns, rows)

	_ = w.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	var bufErr bytes.Buffer
	_, _ = io.Copy(&bufErr, rErr)
	stderrOutput := bufErr.String()

	// Should still have header
	if !strings.Contains(output, "id") {
		t.Error("Output should contain 'id' column header")
	}

	// Should show (0 rows) on stderr
	if !strings.Contains(stderrOutput, "(0 rows)") {
		t.Errorf("Stderr should contain '(0 rows)', got %q", stderrOutput)
	}
}

// TestOutputTableNoColumns tests table output with no columns
func TestOutputTableNoColumns(t *testing.T) {
	columns := []string{}
	rows := [][]string{}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputTable(columns, rows)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should be empty
	if output != "" {
		t.Errorf("Output should be empty for no columns, got %q", output)
	}
}
