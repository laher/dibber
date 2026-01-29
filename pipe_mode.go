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

// runPipeMode reads a query from stdin, executes it, and outputs results to stdout
func runPipeMode(db *sql.DB, format string) {
	// Read all of stdin
	query, err := io.ReadAll(bufio.NewReader(os.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
		os.Exit(1)
	}

	queryStr := strings.TrimSpace(string(query))
	if queryStr == "" {
		fmt.Fprintln(os.Stderr, "Error: No query provided via stdin")
		os.Exit(1)
	}

	// Remove trailing semicolon if present (some drivers don't like it)
	queryStr = strings.TrimSuffix(queryStr, ";")

	// Execute the query
	rows, err := db.Query(queryStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = rows.Close() }()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting columns: %v\n", err)
		os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
			os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error iterating rows: %v\n", err)
		os.Exit(1)
	}

	// Output based on format
	switch strings.ToLower(format) {
	case "csv":
		outputCSV(columns, allRows, ",")
	case "tsv":
		outputCSV(columns, allRows, "\t")
	default:
		outputTable(columns, allRows)
	}
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
