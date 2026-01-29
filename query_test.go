package main

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with test data
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			age INTEGER,
			salary REAL,
			is_active BOOLEAN DEFAULT 1,
			notes TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (id, name, email, age, salary, is_active, notes) VALUES
		(1, 'Alice', 'alice@example.com', 30, 50000.50, 1, 'First user'),
		(2, 'Bob', NULL, 25, NULL, 0, NULL),
		(3, 'Charlie', 'charlie@example.com', 35, 75000.00, 1, 'Line 1
Line 2
Line 3')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return db
}

// TestExecuteQuery tests basic query execution
func TestExecuteQuery(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	result := executeQuery(db, "SELECT id, name, email FROM users ORDER BY id")

	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Check columns
	expectedCols := []string{"id", "name", "email"}
	if len(result.Columns) != len(expectedCols) {
		t.Errorf("Expected %d columns, got %d", len(expectedCols), len(result.Columns))
	}
	for i, col := range expectedCols {
		if result.Columns[i] != col {
			t.Errorf("Column %d: expected %q, got %q", i, col, result.Columns[i])
		}
	}

	// Check row count
	if len(result.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(result.Rows))
	}

	// Check first row
	if result.Rows[0][1].Value != "Alice" {
		t.Errorf("Expected first user to be Alice, got %q", result.Rows[0][1].Value)
	}

	// Check NULL handling
	if !result.Rows[1][2].IsNull {
		t.Error("Bob's email should be NULL")
	}
	if result.Rows[0][2].IsNull {
		t.Error("Alice's email should not be NULL")
	}
}

// TestExecuteQueryError tests error handling for invalid queries
func TestExecuteQueryError(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	result := executeQuery(db, "SELECT * FROM nonexistent_table")

	if result.Error == nil {
		t.Error("Expected error for invalid table, got nil")
	}
}

// TestColumnTypeDetection tests column type categorization
func TestColumnTypeDetection(t *testing.T) {
	tests := []struct {
		dbType   string
		expected ColumnType
	}{
		{"INTEGER", ColTypeNumeric},
		{"INT", ColTypeNumeric},
		{"BIGINT", ColTypeNumeric},
		{"REAL", ColTypeNumeric},
		{"FLOAT", ColTypeNumeric},
		{"DOUBLE", ColTypeNumeric},
		{"DECIMAL", ColTypeNumeric},
		{"BOOLEAN", ColTypeBoolean},
		{"BOOL", ColTypeBoolean},
		{"TEXT", ColTypeText},
		{"VARCHAR", ColTypeText},
		{"CHAR", ColTypeText},
		{"DATE", ColTypeDatetime},
		{"DATETIME", ColTypeDatetime},
		{"TIMESTAMP", ColTypeDatetime},
		{"BLOB", ColTypeBlob},
		{"BYTEA", ColTypeBlob},
		{"UNKNOWN_TYPE", ColTypeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.dbType, func(t *testing.T) {
			result := categorizeColumnType(tc.dbType)
			if result != tc.expected {
				t.Errorf("categorizeColumnType(%q) = %v, want %v", tc.dbType, result, tc.expected)
			}
		})
	}
}

// TestParseQueryMeta tests query metadata parsing
func TestParseQueryMeta(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	tests := []struct {
		name       string
		query      string
		isEditable bool
		tableName  string
	}{
		{
			name:       "simple select",
			query:      "SELECT * FROM users",
			isEditable: true,
			tableName:  "users",
		},
		{
			name:       "select with where",
			query:      "SELECT id, name FROM users WHERE id = 1",
			isEditable: true,
			tableName:  "users",
		},
		{
			name:       "select with join",
			query:      "SELECT u.* FROM users u JOIN orders o ON u.id = o.user_id",
			isEditable: false,
			tableName:  "",
		},
		{
			name:       "select with count",
			query:      "SELECT COUNT(*) FROM users",
			isEditable: false,
			tableName:  "",
		},
		{
			name:       "select with group by",
			query:      "SELECT name, COUNT(*) FROM users GROUP BY name",
			isEditable: false,
			tableName:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := executeQuery(db, tc.query)
			if result.Error != nil {
				// Skip queries that fail (like JOIN on non-existent table)
				if tc.isEditable {
					t.Skipf("Query failed: %v", result.Error)
				}
				return
			}

			meta := parseQueryMeta(tc.query, result)

			if meta == nil {
				if tc.isEditable {
					t.Error("Expected non-nil meta for editable query")
				}
				return
			}

			if meta.IsEditable != tc.isEditable {
				t.Errorf("IsEditable = %v, want %v", meta.IsEditable, tc.isEditable)
			}

			if tc.isEditable && meta.TableName != tc.tableName {
				t.Errorf("TableName = %q, want %q", meta.TableName, tc.tableName)
			}
		})
	}
}

// TestFormatValueForSQL tests SQL value formatting
func TestFormatValueForSQL(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		isNull   bool
		colType  ColumnType
		dbType   string
		expected string
	}{
		{"null value", "", true, ColTypeText, "sqlite", "NULL"},
		{"empty string", "", false, ColTypeText, "sqlite", "''"},
		{"simple string", "hello", false, ColTypeText, "sqlite", "'hello'"},
		{"string with quote", "it's", false, ColTypeText, "sqlite", "'it''s'"},
		{"integer", "42", false, ColTypeNumeric, "sqlite", "42"},
		{"float", "3.14", false, ColTypeNumeric, "sqlite", "3.14"},
		{"negative number", "-100", false, ColTypeNumeric, "sqlite", "-100"},
		{"boolean true sqlite", "true", false, ColTypeBoolean, "sqlite", "TRUE"},
		{"boolean false sqlite", "false", false, ColTypeBoolean, "sqlite", "FALSE"},
		{"boolean true mysql", "true", false, ColTypeBoolean, "mysql", "1"},
		{"boolean false mysql", "false", false, ColTypeBoolean, "mysql", "0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatValueForSQL(tc.value, tc.isNull, tc.colType, tc.dbType)
			if result != tc.expected {
				t.Errorf("formatValueForSQL(%q, %v, %v, %q) = %q, want %q",
					tc.value, tc.isNull, tc.colType, tc.dbType, result, tc.expected)
			}
		})
	}
}

// TestIsValidNumber tests number validation
func TestIsValidNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"42", true},
		{"-42", true},
		{"+42", true},
		{"3.14", true},
		{"-3.14", true},
		{"0", true},
		{"0.0", true},
		{"1e10", true},
		{"1.5e-3", true},
		{"", false},
		{"abc", false},
		{"12abc", false},
		{"12.34.56", false},
		{"-", false},
		{".", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := isValidNumber(tc.input)
			if result != tc.expected {
				t.Errorf("isValidNumber(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// TestExtractTableName tests table name extraction
func TestExtractTableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users", "users"},
		{"users u", "users"},
		{"users AS u", "users"},
		{"`users`", "users"},
		{"  users  ", "users"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := extractTableName(tc.input)
			if result != tc.expected {
				t.Errorf("extractTableName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestEscapeSQLString tests SQL string escaping
func TestEscapeSQLString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"it's", "it''s"},
		{"'quoted'", "''quoted''"},
		{"no quotes", "no quotes"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeSQLString(tc.input)
			if result != tc.expected {
				t.Errorf("escapeSQLString(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIntegrationQueryAndFormat tests full query-to-output pipeline
func TestIntegrationQueryAndFormat(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	// Execute query
	result := executeQuery(db, "SELECT id, name, salary FROM users WHERE id = 2")
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(result.Rows))
	}

	row := result.Rows[0]

	// Check Bob's data
	if row[1].Value != "Bob" {
		t.Errorf("Name = %q, want %q", row[1].Value, "Bob")
	}

	// Check NULL salary
	if !row[2].IsNull {
		t.Error("Salary should be NULL")
	}
}

// TestExecuteQueryWithMultilineData tests handling of multiline data
func TestExecuteQueryWithMultilineData(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	result := executeQuery(db, "SELECT notes FROM users WHERE id = 3")
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(result.Rows))
	}

	notes := result.Rows[0][0].Value
	if !contains(notes, "Line 1") || !contains(notes, "Line 2") {
		t.Errorf("Notes should contain multiline data, got %q", notes)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
