package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dsn := flag.String("dsn", "", "Database connection string")
	dbType := flag.String("type", "", "Database type: mysql, postgres, sqlite (auto-detected if not specified)")
	sqlFile := flag.String("sql-file", "dibber.sql", "SQL file to sync with the query window")
	outputFormat := flag.String("format", "table", "Output format for piped queries: table, csv, tsv")
	flag.Parse()

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "Error: -dsn flag is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  dibber -dsn 'connection_string' [-type mysql|postgres|sqlite] [-sql-file filename.sql]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Interactive mode:")
		fmt.Fprintln(os.Stderr, "  dibber -dsn 'user:password@tcp(localhost:3306)/dbname'")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Pipe mode (query via stdin):")
		fmt.Fprintln(os.Stderr, "  echo 'SELECT * FROM users' | dibber -dsn '...'")
		fmt.Fprintln(os.Stderr, "  cat query.sql | dibber -dsn '...' -format csv")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  -dsn       Database connection string (required)")
		fmt.Fprintln(os.Stderr, "  -type      Database type: mysql, postgres, sqlite (auto-detected)")
		fmt.Fprintln(os.Stderr, "  -sql-file  SQL file to sync queries in interactive mode (default: dibber.sql)")
		fmt.Fprintln(os.Stderr, "  -format    Output format for pipe mode: table, csv, tsv (default: table)")
		os.Exit(1)
	}

	// Auto-detect database type if not specified
	detectedType := *dbType
	if detectedType == "" {
		detectedType = detectDBType(*dsn)
	}

	if detectedType == "" {
		fmt.Fprintln(os.Stderr, "Error: Could not auto-detect database type. Please specify -type flag.")
		os.Exit(1)
	}

	// Map type to driver name
	driverName := getDriverName(detectedType)
	if driverName == "" {
		fmt.Fprintf(os.Stderr, "Error: Unknown database type '%s'. Use mysql, postgres, or sqlite.\n", detectedType)
		os.Exit(1)
	}

	db, err := sql.Open(driverName, *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	// Verify connection
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	// Check if stdin is a pipe (not a terminal)
	if isPiped() {
		// Pipe mode: read query from stdin, execute, output to stdout
		runPipeMode(db, *outputFormat)
		return
	}

	// Interactive mode: start the Bubble Tea UI
	// Load initial SQL content from file (if it exists)
	var initialSQL string
	if data, err := os.ReadFile(*sqlFile); err == nil {
		initialSQL = string(data)
	}

	p := tea.NewProgram(NewModel(db, detectedType, *sqlFile, initialSQL), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// detectDBType attempts to determine the database type from the DSN
func detectDBType(dsn string) string {
	dsnLower := strings.ToLower(dsn)

	// PostgreSQL patterns
	if strings.HasPrefix(dsnLower, "postgres://") ||
		strings.HasPrefix(dsnLower, "postgresql://") ||
		strings.Contains(dsnLower, "host=") {
		return "postgres"
	}

	// MySQL patterns (user:pass@tcp or user:pass@unix)
	if strings.Contains(dsn, "@tcp(") ||
		strings.Contains(dsn, "@unix(") ||
		strings.Contains(dsnLower, "mysql://") {
		return "mysql"
	}

	// SQLite patterns (file path or :memory:)
	if strings.HasSuffix(dsnLower, ".db") ||
		strings.HasSuffix(dsnLower, ".sqlite") ||
		strings.HasSuffix(dsnLower, ".sqlite3") ||
		dsnLower == ":memory:" ||
		strings.HasPrefix(dsn, "/") ||
		strings.HasPrefix(dsn, "./") ||
		strings.HasPrefix(dsn, "file:") {
		return "sqlite"
	}

	return ""
}

// getDriverName returns the SQL driver name for the database type
func getDriverName(dbType string) string {
	switch strings.ToLower(dbType) {
	case "mysql":
		return "mysql"
	case "postgres", "postgresql", "pg":
		return "pgx"
	case "sqlite", "sqlite3":
		return "sqlite3"
	default:
		return ""
	}
}
