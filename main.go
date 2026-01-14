package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	dsn := flag.String("dsn", "", "MySQL DSN (e.g., user:password@tcp(localhost:3306)/dbname)")
	flag.Parse()

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "Error: -dsn flag is required")
		fmt.Fprintln(os.Stderr, "Usage: dabble -dsn 'user:password@tcp(localhost:3306)/dbname'")
		os.Exit(1)
	}

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(NewModel(db), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
