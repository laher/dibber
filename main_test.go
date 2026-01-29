package main

import "testing"

// TestDetectDBType tests database type detection from DSN
func TestDetectDBType(t *testing.T) {
	tests := []struct {
		dsn      string
		expected string
	}{
		{"postgres://user:pass@localhost/db", "postgres"},
		{"postgresql://user:pass@localhost/db", "postgres"},
		{"host=localhost user=test", "postgres"},
		{"user:pass@tcp(localhost:3306)/db", "mysql"},
		{"user:pass@unix(/tmp/mysql.sock)/db", "mysql"},
		{"/path/to/database.db", "sqlite"},
		{"./local.db", "sqlite"},
		{":memory:", "sqlite"},
		{"file:test.db", "sqlite"},
		{"something_unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.dsn, func(t *testing.T) {
			result := detectDBType(tc.dsn)
			if result != tc.expected {
				t.Errorf("detectDBType(%q) = %q, want %q", tc.dsn, result, tc.expected)
			}
		})
	}
}

// TestGetDriverName tests driver name mapping
func TestGetDriverName(t *testing.T) {
	tests := []struct {
		dbType   string
		expected string
	}{
		{"mysql", "mysql"},
		{"postgres", "pgx"},
		{"postgresql", "pgx"},
		{"pg", "pgx"},
		{"sqlite", "sqlite3"},
		{"sqlite3", "sqlite3"},
		{"unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.dbType, func(t *testing.T) {
			result := getDriverName(tc.dbType)
			if result != tc.expected {
				t.Errorf("getDriverName(%q) = %q, want %q", tc.dbType, result, tc.expected)
			}
		})
	}
}
