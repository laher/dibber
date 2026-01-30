package main

import (
	"bufio"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
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
	dsn := flag.String("dsn", "", "Database connection string (use this OR -connection)")
	connectionName := flag.String("connection", "", "Named connection from ~/.dibber.yaml")
	dbType := flag.String("type", "", "Database type: mysql, postgres, sqlite (auto-detected if not specified)")

	// Connection management flags
	addConnection := flag.String("add-connection", "", "Add a new named connection (requires -dsn)")
	removeConnection := flag.String("remove-connection", "", "Remove a saved connection")
	listConnections := flag.Bool("list-connections", false, "List all saved connections")
	listThemes := flag.Bool("list-themes", false, "List all available themes")
	changePassword := flag.Bool("change-password", false, "Change the master password")
	themeName := flag.String("theme", "", "Theme for the connection (use with -add-connection)")

	// Other flags
	sqlFile := flag.String("sql-file", "dibber.sql", "SQL file to sync with the query window")
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
		handleAddConnection(*addConnection, *dsn, *dbType, *themeName)
		return
	}

	if *changePassword {
		handleChangePassword()
		return
	}

	// Determine DSN from either -dsn or -connection
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
	// Load initial SQL content from file (if it exists)
	var initialSQL string
	if data, err := os.ReadFile(*sqlFile); err == nil {
		initialSQL = string(data)
	}

	// Create vault manager for connection switching
	vm := NewVaultManager()
	_ = vm.LoadConfig() // Ignore error - might not have a config yet

	// Get the theme
	theme := GetTheme(connInfo.theme)

	p := tea.NewProgram(NewModel(db, detectedType, *sqlFile, initialSQL, vm, *connectionName, theme), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  dibber -dsn 'connection_string' [-type mysql|postgres|sqlite]")
	fmt.Fprintln(os.Stderr, "  dibber -connection 'name'       (use a saved connection)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Connection Management:")
	fmt.Fprintln(os.Stderr, "  dibber -add-connection 'name' -dsn 'connection_string' [-type db_type]")
	fmt.Fprintln(os.Stderr, "  dibber -remove-connection 'name'")
	fmt.Fprintln(os.Stderr, "  dibber -list-connections")
	fmt.Fprintln(os.Stderr, "  dibber -change-password")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Interactive mode:")
	fmt.Fprintln(os.Stderr, "  dibber -dsn 'user:password@tcp(localhost:3306)/dbname'")
	fmt.Fprintln(os.Stderr, "  dibber -connection prod")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Pipe mode (query via stdin):")
	fmt.Fprintln(os.Stderr, "  echo 'SELECT * FROM users' | dibber -dsn '...'")
	fmt.Fprintln(os.Stderr, "  cat query.sql | dibber -connection prod -format csv")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -dsn             Database connection string")
	fmt.Fprintln(os.Stderr, "  -connection      Named connection from ~/.dibber.yaml")
	fmt.Fprintln(os.Stderr, "  -type            Database type: mysql, postgres, sqlite (auto-detected)")
	fmt.Fprintln(os.Stderr, "  -sql-file        SQL file to sync queries (default: dibber.sql)")
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

		if !vm.HasVault() {
			return connectionInfo{}, errors.New("no saved connections - add one with -add-connection first")
		}

		// Prompt for password to unlock
		password, err := promptPassword("Enter master password: ")
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

	return connectionInfo{}, errors.New("either -dsn or -connection is required")
}

// handleListConnections lists all saved connections
func handleListConnections() {
	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "No configuration file found.")
		return
	}

	if !vm.HasVault() {
		fmt.Fprintln(os.Stderr, "No saved connections.")
		return
	}

	names := vm.ListConnections()
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "No saved connections.")
		return
	}

	fmt.Println("Saved connections:")
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
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
	fmt.Println("Use with -add-connection: dibber -add-connection mydb -dsn '...' -theme dracula")
}

// handleAddConnection adds a new connection
func handleAddConnection(name, dsn, dbType, theme string) {
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

	if vm.HasVault() {
		// Vault exists, unlock it
		password, err := promptPassword("Enter master password: ")
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

	// Auto-detect type if not specified
	if dbType == "" {
		dbType = detectDBType(dsn)
	}

	if err := vm.AddConnection(name, dsn, dbType, theme); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to add connection: %v\n", err)
		os.Exit(1)
	}

	themeInfo := ""
	if theme != "" {
		themeInfo = fmt.Sprintf(" with theme %q", theme)
	}
	fmt.Printf("Connection %q saved successfully%s.\n", name, themeInfo)
}

// handleRemoveConnection removes a connection
func handleRemoveConnection(name string) {
	vm := NewVaultManager()
	if err := vm.LoadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "No configuration file found.")
		os.Exit(1)
	}

	if !vm.HasVault() {
		fmt.Fprintln(os.Stderr, "No saved connections.")
		os.Exit(1)
	}

	password, err := promptPassword("Enter master password: ")
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

// handleChangePassword changes the master password
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
	currentPassword, err := promptPassword("Enter current master password: ")
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
	fmt.Print("Enter new master password: ")
	password1, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}

	if len(password1) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	fmt.Print("Confirm master password: ")
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

// promptLine prompts for a single line of input
func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
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
