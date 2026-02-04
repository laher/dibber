package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/term"
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

	p := tea.NewProgram(NewModel(db, detectedType, resolvedSQLDir, resolvedSQLFile, initialSQL, vm, *connectionName, theme), tea.WithAltScreen())
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

// connectionInfo holds resolved connection details
type connectionInfo struct {
	dsn    string
	dbType string
	theme  string
}

// resolveDSN gets the DSN either directly or from a saved connection
func resolveDSN(dsn, connectionName, dbType string) (connectionInfo, error) {
	// If DSN provided directly, use it
	if dsn != "" {
		return connectionInfo{dsn: dsn, dbType: dbType}, nil
	}

	// If connection name provided, look it up
	if connectionName != "" {
		vm := NewVaultManager()
		if err := vm.LoadConfig(); err != nil {
			return connectionInfo{}, fmt.Errorf("failed to load config: %w", err)
		}

		// Check if this specific connection exists
		if !vm.config.HasConnection(connectionName) {
			return connectionInfo{}, fmt.Errorf("connection %q not found", connectionName)
		}

		// Check if this is a plaintext connection (no password needed)
		if vm.IsPlaintextConnection(connectionName) {
			connDSN, connType, connTheme, err := vm.GetConnection(connectionName)
			if err != nil {
				return connectionInfo{}, fmt.Errorf("connection %q not found", connectionName)
			}

			// Use stored type if not overridden
			if dbType == "" {
				dbType = connType
			}

			return connectionInfo{dsn: connDSN, dbType: dbType, theme: connTheme}, nil
		}

		// Encrypted connection - need vault
		if !vm.HasVault() {
			return connectionInfo{}, errors.New("no encrypted connections configured - connection may be corrupted")
		}

		// Prompt for password to unlock
		password, err := promptPassword("Enter encryption password: ")
		if err != nil {
			return connectionInfo{}, fmt.Errorf("failed to read password: %w", err)
		}

		if err := vm.Unlock(password); err != nil {
			if errors.Is(err, ErrDecryptionFailed) {
				return connectionInfo{}, errors.New("incorrect password")
			}
			return connectionInfo{}, fmt.Errorf("failed to unlock vault: %w", err)
		}

		connDSN, connType, connTheme, err := vm.GetConnection(connectionName)
		if err != nil {
			return connectionInfo{}, fmt.Errorf("connection %q not found", connectionName)
		}

		// Use stored type if not overridden
		if dbType == "" {
			dbType = connType
		}

		return connectionInfo{dsn: connDSN, dbType: dbType, theme: connTheme}, nil
	}

	return connectionInfo{}, errors.New("either -dsn or -conn is required")
}

// handleListConnections lists all saved connections
func handleListConnections() {
	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "No configuration file found.")
		return
	}

	names := vm.ListConnections()
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "No saved connections.")
		return
	}

	fmt.Println("Saved connections:")
	for _, name := range names {
		encStatus := ""
		if vm.IsPlaintextConnection(name) {
			encStatus = " (plaintext)"
		} else {
			encStatus = " (encrypted)"
		}
		fmt.Printf("  - %s%s\n", name, encStatus)
	}
}

// handleListThemes lists all available themes
func handleListThemes() {
	fmt.Println("Available themes:")
	for _, name := range ThemeNames() {
		theme := Themes[name]
		fmt.Printf("  - %-14s %s\n", name, theme.Description)
	}
	fmt.Println()
	fmt.Println("Use with -add-conn: dibber -add-conn mydb -dsn '...' -theme dracula")
}

// handleAddConnection adds a new connection
func handleAddConnection(name, dsn, dbType, theme string, noEncrypt bool) {
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "Error: -dsn is required when adding a connection")
		os.Exit(1)
	}

	// Validate theme if specified
	if theme != "" {
		if _, ok := Themes[theme]; !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown theme %q. Use -list-themes to see available themes.\n", theme)
			os.Exit(1)
		}
	}

	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil && !errors.Is(err, ErrConfigNotFound) {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Auto-detect type if not specified
	if dbType == "" {
		dbType = detectDBType(dsn)
	}

	if noEncrypt {
		// Store plaintext connection - no vault needed
		if err := vm.AddConnectionWithEncryption(name, dsn, dbType, theme, false); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to add connection: %v\n", err)
			os.Exit(1)
		}

		themeInfo := ""
		if theme != "" {
			themeInfo = fmt.Sprintf(" with theme %q", theme)
		}
		fmt.Printf("Connection %q saved (plaintext)%s.\n", name, themeInfo)
		return
	}

	// Encrypted connection - need vault
	if vm.HasVault() {
		// Vault exists, unlock it
		password, err := promptPassword("Enter encryption password: ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read password: %v\n", err)
			os.Exit(1)
		}

		if err := vm.Unlock(password); err != nil {
			if errors.Is(err, ErrDecryptionFailed) {
				fmt.Fprintln(os.Stderr, "Incorrect password.")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Failed to unlock vault: %v\n", err)
			os.Exit(1)
		}
	} else {
		// First time - create new vault
		fmt.Println("Creating new encrypted connection store...")
		password, err := promptNewPassword()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set password: %v\n", err)
			os.Exit(1)
		}

		if err := vm.InitializeWithPassword(password); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize vault: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Vault initialized successfully.")
	}

	if err := vm.AddConnection(name, dsn, dbType, theme); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to add connection: %v\n", err)
		os.Exit(1)
	}

	themeInfo := ""
	if theme != "" {
		themeInfo = fmt.Sprintf(" with theme %q", theme)
	}
	fmt.Printf("Connection %q saved (encrypted)%s.\n", name, themeInfo)
}

// handleRemoveConnection removes a connection
func handleRemoveConnection(name string) {
	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "No configuration file found.")
		os.Exit(1)
	}

	if !vm.config.HasConnection(name) {
		fmt.Fprintf(os.Stderr, "Connection %q not found.\n", name)
		os.Exit(1)
	}

	// Check if it's a plaintext connection (no password needed)
	if vm.IsPlaintextConnection(name) {
		if err := vm.RemovePlaintextConnection(name); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove connection: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Connection %q removed.\n", name)
		return
	}

	// Encrypted connection - need vault password
	if !vm.HasVault() {
		fmt.Fprintln(os.Stderr, "No encrypted vault configured.")
		os.Exit(1)
	}

	password, err := promptPassword("Enter encryption password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read password: %v\n", err)
		os.Exit(1)
	}

	if err := vm.Unlock(password); err != nil {
		if errors.Is(err, ErrDecryptionFailed) {
			fmt.Fprintln(os.Stderr, "Incorrect password.")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Failed to unlock vault: %v\n", err)
		os.Exit(1)
	}

	if err := vm.RemoveConnection(name); err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			fmt.Fprintf(os.Stderr, "Connection %q not found.\n", name)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Failed to remove connection: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connection %q removed.\n", name)
}

// handleSetSQLDir sets the SQL directory in the config
func handleSetSQLDir(dir string) {
	// Expand ~ to home directory
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
			os.Exit(1)
		}
		dir = filepath.Join(home, dir[2:])
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve path: %v\n", err)
		os.Exit(1)
	}

	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil && !errors.Is(err, ErrConfigNotFound) {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := vm.SetSQLDir(absDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("SQL directory set to: %s\n", absDir)
}

// handleChangePassword changes the encryption password
func handleChangePassword() {
	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "No configuration file found.")
		os.Exit(1)
	}

	if !vm.HasVault() {
		fmt.Fprintln(os.Stderr, "No vault to change password for.")
		os.Exit(1)
	}

	// Unlock with current password
	currentPassword, err := promptPassword("Enter current encryption password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read password: %v\n", err)
		os.Exit(1)
	}

	if err := vm.Unlock(currentPassword); err != nil {
		if errors.Is(err, ErrDecryptionFailed) {
			fmt.Fprintln(os.Stderr, "Incorrect password.")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Failed to unlock vault: %v\n", err)
		os.Exit(1)
	}

	// Get new password
	newPassword, err := promptNewPassword()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set new password: %v\n", err)
		os.Exit(1)
	}

	if err := vm.ChangePassword(newPassword); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change password: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Password changed successfully.")
}

// promptPassword prompts for a password without echo
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Newline after password entry
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// promptNewPassword prompts for a new password with confirmation
func promptNewPassword() (string, error) {
	fmt.Print("Enter new encryption password: ")
	password1, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}

	if len(password1) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	fmt.Print("Confirm encryption password: ")
	password2, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}

	if string(password1) != string(password2) {
		return "", errors.New("passwords do not match")
	}

	return string(password1), nil
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

// extractDatabaseName extracts the database/schema name from a DSN
// Returns the database name if found, or a fallback based on dbType
func extractDatabaseName(dsn, dbType string) string {
	dbType = strings.ToLower(dbType)

	switch dbType {
	case "postgres", "postgresql", "pg":
		// PostgreSQL URL format: postgres://user:pass@host:port/database?params
		// or key=value format: host=localhost dbname=mydb
		if strings.HasPrefix(strings.ToLower(dsn), "postgres://") ||
			strings.HasPrefix(strings.ToLower(dsn), "postgresql://") {
			// URL format - extract path component
			// Remove query string first
			dsnClean := dsn
			if idx := strings.Index(dsnClean, "?"); idx != -1 {
				dsnClean = dsnClean[:idx]
			}
			// Find the last /
			if idx := strings.LastIndex(dsnClean, "/"); idx != -1 {
				dbName := dsnClean[idx+1:]
				if dbName != "" {
					return sanitizeFilename(dbName)
				}
			}
		} else if strings.Contains(dsn, "dbname=") {
			// Key=value format
			parts := strings.Fields(dsn)
			for _, part := range parts {
				if strings.HasPrefix(part, "dbname=") {
					dbName := strings.TrimPrefix(part, "dbname=")
					if dbName != "" {
						return sanitizeFilename(dbName)
					}
				}
			}
		}
		return "postgres"

	case "mysql":
		// MySQL format: user:pass@tcp(host:port)/database?params
		// or user:pass@unix(/path/to/socket)/database
		// Find the database name after the last /
		dsnClean := dsn
		if idx := strings.Index(dsnClean, "?"); idx != -1 {
			dsnClean = dsnClean[:idx]
		}
		if idx := strings.LastIndex(dsnClean, "/"); idx != -1 {
			dbName := dsnClean[idx+1:]
			if dbName != "" {
				return sanitizeFilename(dbName)
			}
		}
		return "mysql"

	case "sqlite", "sqlite3":
		// SQLite: /path/to/file.db or :memory: or file:path?params
		if dsn == ":memory:" {
			return "memory"
		}
		dsnClean := dsn
		// Remove file: prefix
		dsnClean = strings.TrimPrefix(dsnClean, "file:")
		// Remove query string
		if idx := strings.Index(dsnClean, "?"); idx != -1 {
			dsnClean = dsnClean[:idx]
		}
		// Get the base filename without extension
		base := filepath.Base(dsnClean)
		// Remove common extensions
		for _, ext := range []string{".sqlite3", ".sqlite", ".db"} {
			if strings.HasSuffix(strings.ToLower(base), ext) {
				base = base[:len(base)-len(ext)]
				break
			}
		}
		if base != "" && base != "." {
			return sanitizeFilename(base)
		}
		return "sqlite"
	}

	// Unknown type - use type name or fallback
	if dbType != "" {
		return dbType
	}
	return "dibber"
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
