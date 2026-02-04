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

// TestExtractDatabaseName tests database name extraction from DSN
func TestExtractDatabaseName(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		dbType   string
		expected string
	}{
		// PostgreSQL URL format
		{"postgres url simple", "postgres://user:pass@localhost/mydb", "postgres", "mydb"},
		{"postgres url with port", "postgres://user:pass@localhost:5432/proddb", "postgres", "proddb"},
		{"postgres url with params", "postgres://user:pass@localhost/testdb?sslmode=disable", "postgres", "testdb"},
		{"postgresql url", "postgresql://user:pass@localhost/appdb", "postgres", "appdb"},
		{"postgres url no db", "postgres://user:pass@localhost/", "postgres", "postgres"},

		// PostgreSQL key=value format
		{"postgres kv format", "host=localhost dbname=mydb user=test", "postgres", "mydb"},
		{"postgres kv no dbname", "host=localhost user=test", "postgres", "postgres"},

		// MySQL format
		{"mysql tcp", "user:pass@tcp(localhost:3306)/myapp", "mysql", "myapp"},
		{"mysql tcp with params", "user:pass@tcp(localhost:3306)/orders?parseTime=true", "mysql", "orders"},
		{"mysql unix socket", "user:pass@unix(/tmp/mysql.sock)/inventory", "mysql", "inventory"},
		{"mysql no db", "user:pass@tcp(localhost:3306)/", "mysql", "mysql"},

		// SQLite format
		{"sqlite file path", "/path/to/mydata.db", "sqlite", "mydata"},
		{"sqlite relative", "./local.sqlite3", "sqlite", "local"},
		{"sqlite memory", ":memory:", "sqlite", "memory"},
		{"sqlite file uri", "file:test.db?cache=shared", "sqlite", "test"},
		{"sqlite with sqlite ext", "/data/app.sqlite", "sqlite", "app"},

		// Edge cases
		{"special chars in name", "postgres://user:pass@localhost/my-db_v2", "postgres", "my-db_v2"},
		{"unknown type", "something://unknown", "unknown", "unknown"},
		{"empty type fallback", "something://unknown", "", "dibber"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractDatabaseName(tc.dsn, tc.dbType)
			if result != tc.expected {
				t.Errorf("extractDatabaseName(%q, %q) = %q, want %q", tc.dsn, tc.dbType, result, tc.expected)
			}
		})
	}
}

// TestSanitizeFilename tests filename sanitization
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"with/slash", "with_slash"},
		{"with:colon", "with_colon"},
		{"multiple   spaces", "multiple___spaces"},
		{"", "dibber"},
		{"___", "dibber"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			if result != tc.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
