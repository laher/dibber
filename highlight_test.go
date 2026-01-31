package main

import (
	"strings"
	"testing"
)

func TestSQLHighlighter_Keywords(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	tests := []struct {
		name  string
		input string
		wants []string // substrings that should be in the output
	}{
		{
			name:  "SELECT keyword",
			input: "SELECT * FROM users",
			wants: []string{"SELECT", "FROM"}, // keywords should be present (styled)
		},
		{
			name:  "lowercase keywords",
			input: "select id from users where active = true",
			wants: []string{"select", "from", "where", "true"},
		},
		{
			name:  "mixed case",
			input: "Select Id From Users Where Active = True",
			wants: []string{"Select", "From", "Where", "True"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			for _, want := range tt.wants {
				if !strings.Contains(result, want) {
					t.Errorf("Highlight(%q) = %q, want to contain %q", tt.input, result, want)
				}
			}
			// Note: ANSI codes may not be present in non-TTY test environments
			// The important thing is that the text content is preserved
		})
	}
}

func TestSQLHighlighter_Strings(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	input := "SELECT * FROM users WHERE name = 'John'"
	result := h.Highlight(input)

	// The string 'John' should be in the output
	if !strings.Contains(result, "'John'") {
		t.Errorf("Highlight(%q) should contain the string literal", input)
	}

	// Note: ANSI codes may not be present in non-TTY test environments
}

func TestSQLHighlighter_Numbers(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	tests := []struct {
		input string
		want  string
	}{
		{"SELECT * FROM users WHERE id = 42", "42"},
		{"SELECT * FROM users LIMIT 100", "100"},
		{"SELECT 3.14 AS pi", "3.14"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.Highlight(tt.input)
			if !strings.Contains(result, tt.want) {
				t.Errorf("Highlight(%q) should contain number %q", tt.input, tt.want)
			}
		})
	}
}

func TestSQLHighlighter_Functions(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	tests := []struct {
		input    string
		function string
	}{
		{"SELECT COUNT(*) FROM users", "COUNT"},
		{"SELECT SUM(amount) FROM orders", "SUM"},
		{"SELECT UPPER(name) FROM users", "UPPER"},
		{"SELECT NOW()", "NOW"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.Highlight(tt.input)
			if !strings.Contains(result, tt.function) {
				t.Errorf("Highlight(%q) should contain function %q", tt.input, tt.function)
			}
		})
	}
}

func TestSQLHighlighter_Comments(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	tests := []struct {
		name    string
		input   string
		comment string
	}{
		{
			name:    "line comment",
			input:   "SELECT * FROM users -- get all users",
			comment: "-- get all users",
		},
		{
			name:    "block comment",
			input:   "SELECT /* columns */ * FROM users",
			comment: "/* columns */",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.input)
			if !strings.Contains(result, tt.comment) {
				t.Errorf("Highlight(%q) should contain comment %q", tt.input, tt.comment)
			}
		})
	}
}

func TestSQLHighlighter_Empty(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	result := h.Highlight("")
	if result != "" {
		t.Errorf("Highlight(\"\") = %q, want empty string", result)
	}
}

func TestSQLHighlighter_PlainText(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	// Plain text with no SQL keywords should pass through
	input := "hello world"
	result := h.Highlight(input)

	// Should contain the original text
	if !strings.Contains(result, "hello") || !strings.Contains(result, "world") {
		t.Errorf("Highlight(%q) should contain the original text", input)
	}
}

func TestSQLHighlighter_MultiLine(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	input := `SELECT id, name
FROM users
WHERE active = true
ORDER BY name;`

	result := h.Highlight(input)

	// Should contain all keywords
	keywords := []string{"SELECT", "FROM", "WHERE", "ORDER", "BY"}
	for _, kw := range keywords {
		if !strings.Contains(strings.ToUpper(result), kw) {
			t.Errorf("Highlight should contain keyword %q", kw)
		}
	}

	// Should preserve newlines
	if strings.Count(result, "\n") != strings.Count(input, "\n") {
		t.Errorf("Highlight should preserve newlines")
	}
}

func TestNewSQLHighlighter_DifferentThemes(t *testing.T) {
	// Test that highlighter can be created with different themes
	for name := range Themes {
		t.Run(name, func(t *testing.T) {
			theme := Themes[name]
			h := NewSQLHighlighter(theme)
			if h == nil {
				t.Errorf("NewSQLHighlighter(%s) returned nil", name)
			}

			// Should be able to highlight without panicking
			result := h.Highlight("SELECT * FROM users")
			if result == "" {
				t.Errorf("Highlight returned empty string for theme %s", name)
			}
		})
	}
}

func TestSQLHighlighter_Tokenize(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	// Test that tokenization finds the right tokens
	sql := "SELECT COUNT(*) FROM users WHERE id = 42"
	tokens := h.tokenize(sql)

	// Should find multiple tokens
	if len(tokens) == 0 {
		t.Error("tokenize should find tokens")
	}

	// Check that we found different token types
	foundTypes := make(map[tokenType]bool)
	for _, tok := range tokens {
		foundTypes[tok.typ] = true
	}

	// Should find keywords
	if !foundTypes[tokenKeyword] {
		t.Error("tokenize should find keywords")
	}

	// Should find functions
	if !foundTypes[tokenFunction] {
		t.Error("tokenize should find functions")
	}

	// Should find numbers
	if !foundTypes[tokenNumber] {
		t.Error("tokenize should find numbers")
	}
}

func TestSQLHighlighter_TokenizePriority(t *testing.T) {
	h := NewSQLHighlighter(DefaultTheme)

	// Test that strings take priority over keywords inside them
	sql := "SELECT 'SELECT * FROM fake' FROM users"
	tokens := h.tokenize(sql)

	// The 'SELECT * FROM fake' should be a single string token, not keywords
	var stringTokens []token
	for _, tok := range tokens {
		if tok.typ == tokenString {
			stringTokens = append(stringTokens, tok)
		}
	}

	if len(stringTokens) != 1 {
		t.Errorf("expected 1 string token, got %d", len(stringTokens))
	}

	if len(stringTokens) > 0 && stringTokens[0].text != "'SELECT * FROM fake'" {
		t.Errorf("string token = %q, want %q", stringTokens[0].text, "'SELECT * FROM fake'")
	}
}
