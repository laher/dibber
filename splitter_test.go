package main

import (
	"reflect"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single statement no semicolon",
			input:    "SELECT * FROM users",
			expected: []string{"SELECT * FROM users"},
		},
		{
			name:     "single statement with semicolon",
			input:    "SELECT * FROM users;",
			expected: []string{"SELECT * FROM users"},
		},
		{
			name:     "two statements",
			input:    "SELECT 1; SELECT 2",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:     "two statements with trailing semicolon",
			input:    "SELECT 1; SELECT 2;",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:     "semicolon in single-quoted string",
			input:    "SELECT 'hello; world' FROM t; SELECT 2",
			expected: []string{"SELECT 'hello; world' FROM t", "SELECT 2"},
		},
		{
			name:     "semicolon in double-quoted string",
			input:    `SELECT "col;name" FROM t; SELECT 2`,
			expected: []string{`SELECT "col;name" FROM t`, "SELECT 2"},
		},
		{
			name:     "escaped single quote",
			input:    "SELECT 'it''s a test; really'; SELECT 2",
			expected: []string{"SELECT 'it''s a test; really'", "SELECT 2"},
		},
		{
			name:     "backslash escaped quote (MySQL style)",
			input:    "SELECT 'it\\'s a test; really'; SELECT 2",
			expected: []string{"SELECT 'it\\'s a test; really'", "SELECT 2"},
		},
		{
			name:     "line comment with semicolon",
			input:    "SELECT 1; -- this is a comment; with semicolon\nSELECT 2",
			expected: []string{"SELECT 1", "-- this is a comment; with semicolon\nSELECT 2"},
		},
		{
			name:     "block comment with semicolon",
			input:    "SELECT /* comment; here */ 1; SELECT 2",
			expected: []string{"SELECT /* comment; here */ 1", "SELECT 2"},
		},
		{
			name:     "multiline block comment",
			input:    "SELECT /* multi\nline;\ncomment */ 1; SELECT 2",
			expected: []string{"SELECT /* multi\nline;\ncomment */ 1", "SELECT 2"},
		},
		{
			name:     "empty statements ignored",
			input:    "SELECT 1;; ; SELECT 2;",
			expected: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name:     "whitespace only",
			input:    "   \n\t  ",
			expected: nil,
		},
		{
			name:     "complex mixed example",
			input:    "INSERT INTO t VALUES ('a;b'); UPDATE t SET x='y;z' WHERE id=1; SELECT * FROM t",
			expected: []string{"INSERT INTO t VALUES ('a;b')", "UPDATE t SET x='y;z' WHERE id=1", "SELECT * FROM t"},
		},
		{
			name:  "multiline statements",
			input: "SELECT\n  *\nFROM\n  users;\n\nSELECT\n  *\nFROM\n  orders;",
			expected: []string{
				"SELECT\n  *\nFROM\n  users",
				"SELECT\n  *\nFROM\n  orders",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitStatements(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SplitStatements(%q)\n  got:  %#v\n  want: %#v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSelectStatement(t *testing.T) {
	tests := []struct {
		stmt     string
		expected bool
	}{
		{"SELECT * FROM users", true},
		{"select * from users", true},
		{"  SELECT * FROM users", true},
		{"\n\tSELECT 1", true},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", true},
		{"SHOW TABLES", true},
		{"DESCRIBE users", true},
		{"DESC users", true},
		{"EXPLAIN SELECT * FROM users", true},
		{"TABLE users", true},
		{"VALUES (1), (2)", true},
		{"PRAGMA table_info(users)", true},
		{"INSERT INTO users VALUES (1)", false},
		{"UPDATE users SET name='x'", false},
		{"DELETE FROM users", false},
		{"CREATE TABLE t (id INT)", false},
		{"DROP TABLE t", false},
		{"ALTER TABLE t ADD col INT", false},
		{"SELECTIVE", false}, // starts with SELECT but is different word
		{"SHOWING", false},   // starts with SHOW but is different word
	}

	for _, tt := range tests {
		t.Run(tt.stmt, func(t *testing.T) {
			result := IsSelectStatement(tt.stmt)
			if result != tt.expected {
				t.Errorf("IsSelectStatement(%q) = %v, want %v", tt.stmt, result, tt.expected)
			}
		})
	}
}
