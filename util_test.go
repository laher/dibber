package main

import "testing"

// TestQuoteIdentifier tests identifier quoting
func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		dbType   string
		expected string
	}{
		{"mysql", "`"},
		{"postgres", `"`},
		{"sqlite", `"`},
		{"unknown", `"`},
	}

	for _, tc := range tests {
		t.Run(tc.dbType, func(t *testing.T) {
			result := quoteIdentifier(tc.dbType)
			if result != tc.expected {
				t.Errorf("quoteIdentifier(%q) = %q, want %q", tc.dbType, result, tc.expected)
			}
		})
	}
}

// TestTruncateString tests string truncation
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := truncateString(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

// TestPadRight tests string padding
func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"abc", 5, "abc  "},
		{"hello", 5, "hello"},
		{"hi", 2, "hi"},
		{"", 3, "   "},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := padRight(tc.input, tc.length)
			if result != tc.expected {
				t.Errorf("padRight(%q, %d) = %q, want %q",
					tc.input, tc.length, result, tc.expected)
			}
		})
	}
}
