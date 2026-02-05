package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Connection flags
	dsn := flag.String("dsn", "", "Database connection string (use this OR -conn)")
	connectionName := flag.String("conn", "", "Named connection from ~/.dibber.yaml")
	dbType := flag.String("type", "", "Database type: mysql, postgres, sqlite (auto-detected if not specified)")

	// Connection management flags
	addConnection := flag.String("add-conn", "", "Add a new named connection (requires -dsn)")
	removeConnection := flag.String("remove-conn", "", "Remove a saved connection")
	listConnections := flag.Bool("list-conns", false, "List all saved connections")
	listThemes := flag.Bool("list-themes", false, "List all available themes")
	changePassword := flag.Bool("change-password", false, "Change the encryption password")
	themeName := flag.String("theme", "", "Theme for the connection (use with -add-conn)")
	noEncrypt := flag.Bool("no-encrypt", false, "Store DSN in plaintext (use with -add-conn for local databases)")

	// Other flags
	sqlDir := flag.String("sql-dir", "", "Directory for SQL files (overrides config, default: $HOME/sql)")
	setSQLDir := flag.String("set-sql-dir", "", "Set the SQL directory in config")
	sqlFile := flag.String("sql-file", "", "SQL file to sync with the query window (default: derived from database name)")
	outputFormat := flag.String("format", "table", "Output format for piped queries: table, csv, tsv")
	flag.Parse()

	// Handle connection management commands
	if *listThemes {
		handleListThemes()
		return
	}

	if *listConnections {
		handleListConnections()
		return
	}

	if *removeConnection != "" {
		handleRemoveConnection(*removeConnection)
		return
	}

	if *addConnection != "" {
		handleAddConnection(*addConnection, *dsn, *dbType, *themeName, *noEncrypt)
		return
	}

	if *changePassword {
		handleChangePassword()
		return
	}

	if *setSQLDir != "" {
		handleSetSQLDir(*setSQLDir)
		return
	}

	// Determine DSN from either -dsn or -conn
	connInfo, err := resolveDSN(*dsn, *connectionName, *dbType)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		printUsage()
		os.Exit(1)
	}

	// Auto-detect database type if not specified
	detectedType := connInfo.dbType
	if detectedType == "" {
		detectedType = detectDBType(connInfo.dsn)
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

	db, err := sql.Open(driverName, connInfo.dsn)
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

	// Create vault manager for connection switching and config
	vm := NewVaultManager()
	_ = vm.LoadConfig() // Ignore error - might not have a config yet

	// Determine SQL directory: flag overrides config, config overrides default
	resolvedSQLDir := vm.GetSQLDir() // Gets from config or default
	if *sqlDir != "" {
		resolvedSQLDir = *sqlDir // Flag overrides
	}

	// Ensure SQL directory exists
	if err := os.MkdirAll(resolvedSQLDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create SQL directory %s: %v\n", resolvedSQLDir, err)
		os.Exit(1)
	}

	// Resolve SQL file path (relative to sql-dir unless absolute)
	// If not specified, derive from database name
	resolvedSQLFile := *sqlFile
	if resolvedSQLFile == "" {
		dbName := extractDatabaseName(connInfo.dsn, detectedType)
		resolvedSQLFile = dbName + ".sql"
	}
	if !filepath.IsAbs(resolvedSQLFile) {
		resolvedSQLFile = filepath.Join(resolvedSQLDir, resolvedSQLFile)
	}

	// Load initial SQL content from file (if it exists)
	var initialSQL string
	if data, err := os.ReadFile(resolvedSQLFile); err == nil {
		initialSQL = string(data)
	}

	// Get the theme
	theme := GetTheme(connInfo.theme)

	p := tea.NewProgram(NewModel(db, detectedType, resolvedSQLDir, resolvedSQLFile, initialSQL, vm, *connectionName, theme), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  dibber -dsn 'connection_string' [-type mysql|postgres|sqlite]")
	fmt.Fprintln(os.Stderr, "  dibber -conn 'name'       (use a saved connection)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Connection Management:")
	fmt.Fprintln(os.Stderr, "  dibber -add-conn 'name' -dsn 'connection_string' [-type db_type] [-no-encrypt]")
	fmt.Fprintln(os.Stderr, "  dibber -remove-conn 'name'")
	fmt.Fprintln(os.Stderr, "  dibber -list-conns")
	fmt.Fprintln(os.Stderr, "  dibber -change-password")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Interactive mode:")
	fmt.Fprintln(os.Stderr, "  dibber -dsn 'user:password@tcp(localhost:3306)/dbname'")
	fmt.Fprintln(os.Stderr, "  dibber -conn prod")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Pipe mode (query via stdin):")
	fmt.Fprintln(os.Stderr, "  echo 'SELECT * FROM users' | dibber -dsn '...'")
	fmt.Fprintln(os.Stderr, "  cat query.sql | dibber -conn prod -format csv")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -dsn             Database connection string")
	fmt.Fprintln(os.Stderr, "  -conn            Named connection from ~/.dibber.yaml")
	fmt.Fprintln(os.Stderr, "  -type            Database type: mysql, postgres, sqlite (auto-detected)")
	fmt.Fprintln(os.Stderr, "  -no-encrypt      Store DSN in plaintext (for local databases, no password needed)")
	fmt.Fprintln(os.Stderr, "  -sql-dir         Directory for SQL files (overrides config)")
	fmt.Fprintln(os.Stderr, "  -set-sql-dir     Set the SQL directory in config")
	fmt.Fprintln(os.Stderr, "  -sql-file        SQL file to sync queries (default: [database_name].sql)")
	fmt.Fprintln(os.Stderr, "  -format          Output format for pipe mode: table, csv, tsv (default: table)")
}

// sanitizeFilename removes or replaces characters that are problematic in filenames
func sanitizeFilename(name string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	result := replacer.Replace(name)
	// Remove leading/trailing underscores
	result = strings.Trim(result, "_")
	if result == "" {
		return "dibber"
	}
	return result
}
