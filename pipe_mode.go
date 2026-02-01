package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
)

// isPiped returns true if stdin is connected to a pipe rather than a terminal
func isPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a character device (terminal) or not
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// runPipeMode reads queries from stdin, executes them, and outputs results to stdout
func runPipeMode(db *sql.DB, format string) {
	// Read all of stdin
	input, err := io.ReadAll(bufio.NewReader(os.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
		os.Exit(1)
	}

	inputStr := strings.TrimSpace(string(input))
	if inputStr == "" {
		fmt.Fprintln(os.Stderr, "Error: No query provided via stdin")
		os.Exit(1)
	}

	// Split into individual statements
	statements := SplitStatements(inputStr)
	if len(statements) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No valid statements found")
		os.Exit(1)
	}

	// Track if we've output anything (for separating multiple results)
	firstOutput := true
	hasError := false

	for i, stmt := range statements {
		if IsSelectStatement(stmt) {
			// Execute as query (returns rows)
			columns, rows, err := executeSelectStatement(db, stmt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Statement %d error: %v\n", i+1, err)
				hasError = true
				continue
			}

			// Add separator between multiple result sets
			if !firstOutput {
				fmt.Println()
				if format == "table" {
					fmt.Println("---")
					fmt.Println()
				}
			}
			firstOutput = false

			// Output based on format
			switch strings.ToLower(format) {
			case "csv":
				outputCSV(columns, rows, ",")
			case "tsv":
				outputCSV(columns, rows, "\t")
			default:
				outputTable(columns, rows)
			}
		} else {
			// Execute as statement (INSERT/UPDATE/DELETE/DDL)
			affected, err := executeNonSelectStatement(db, stmt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Statement %d error: %v\n", i+1, err)
				hasError = true
				continue
			}

			// Report affected rows to stderr (doesn't interfere with data output)
			if affected >= 0 {
				fmt.Fprintf(os.Stderr, "Statement %d: %d row(s) affected\n", i+1, affected)
			} else {
				fmt.Fprintf(os.Stderr, "Statement %d: OK\n", i+1)
			}
		}
	}

	if hasError {
		os.Exit(1)
	}
}

// executeSelectStatement executes a SELECT query and returns columns and rows
func executeSelectStatement(db *sql.DB, stmt string) ([]string, [][]string, error) {
	rows, err := db.Query(stmt)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting columns: %w", err)
	}

	// Collect all rows
	var allRows [][]string
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("error scanning row: %w", err)
		}

		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		allRows = append(allRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return columns, allRows, nil
}

// executeNonSelectStatement executes an INSERT/UPDATE/DELETE/DDL statement
// Returns the number of affected rows, or -1 if not applicable
func executeNonSelectStatement(db *sql.DB, stmt string) (int64, error) {
	result, err := db.Exec(stmt)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		// Some statements don't support RowsAffected (e.g., DDL)
		return -1, nil
	}

	return affected, nil
}

// outputTable outputs results in a formatted table
func outputTable(columns []string, rows [][]string) {
	if len(columns) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Cap widths at 50 for readability
	for i := range widths {
		if widths[i] > 50 {
			widths[i] = 50
		}
	}

	// Print header
	var header []string
	for i, col := range columns {
		header = append(header, padAndTruncate(col, widths[i]))
	}
	fmt.Println(strings.Join(header, " | "))

	// Print separator
	var sep []string
	for _, w := range widths {
		sep = append(sep, strings.Repeat("-", w))
	}
	fmt.Println(strings.Join(sep, "-+-"))

	// Print rows
	for _, row := range rows {
		var cells []string
		for i, cell := range row {
			cells = append(cells, padAndTruncate(cell, widths[i]))
		}
		fmt.Println(strings.Join(cells, " | "))
	}

	// Print row count to stderr (so it doesn't interfere with piping)
	fmt.Fprintf(os.Stderr, "\n(%d rows)\n", len(rows))
}

// outputCSV outputs results in CSV or TSV format
func outputCSV(columns []string, rows [][]string, delimiter string) {
	// Print header
	fmt.Println(strings.Join(columns, delimiter))

	// Print rows
	for _, row := range rows {
		// Escape fields if they contain the delimiter or quotes
		escaped := make([]string, len(row))
		for i, cell := range row {
			if strings.Contains(cell, delimiter) || strings.Contains(cell, "\"") || strings.Contains(cell, "\n") {
				// Quote the field and escape internal quotes
				cell = strings.ReplaceAll(cell, "\"", "\"\"")
				escaped[i] = "\"" + cell + "\""
			} else {
				escaped[i] = cell
			}
		}
		fmt.Println(strings.Join(escaped, delimiter))
	}
}

// padAndTruncate pads or truncates a string to the specified width
func padAndTruncate(s string, width int) string {
	// Handle newlines - just take the first line
	if idx := strings.Index(s, "\n"); idx != -1 {
		s = s[:idx] + "..."
	}

	if len(s) > width {
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
