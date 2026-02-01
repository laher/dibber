package main

import (
	"strings"
	"unicode"
)

// SplitStatements splits a SQL string into individual statements.
// It respects:
// - Single-quoted strings ('...')
// - Double-quoted strings ("...")
// - Line comments (--)
// - Block comments (/* ... */)
//
// It does NOT handle:
// - PostgreSQL dollar-quoted strings ($$...$$)
// - MySQL DELIMITER command
// - Backtick-quoted identifiers containing semicolons
func SplitStatements(sql string) []string {
	var statements []string
	var current strings.Builder

	i := 0
	n := len(sql)

	for i < n {
		ch := sql[i]

		// Check for line comment (--)
		if ch == '-' && i+1 < n && sql[i+1] == '-' {
			// Consume until end of line
			current.WriteByte(ch)
			i++
			for i < n && sql[i] != '\n' {
				current.WriteByte(sql[i])
				i++
			}
			if i < n {
				current.WriteByte(sql[i]) // include the newline
				i++
			}
			continue
		}

		// Check for block comment (/* ... */)
		if ch == '/' && i+1 < n && sql[i+1] == '*' {
			current.WriteByte(ch)
			i++
			current.WriteByte(sql[i]) // *
			i++
			// Consume until */
			for i < n {
				if sql[i] == '*' && i+1 < n && sql[i+1] == '/' {
					current.WriteByte(sql[i])
					i++
					current.WriteByte(sql[i])
					i++
					break
				}
				current.WriteByte(sql[i])
				i++
			}
			continue
		}

		// Check for single-quoted string
		if ch == '\'' {
			current.WriteByte(ch)
			i++
			for i < n {
				if sql[i] == '\'' {
					current.WriteByte(sql[i])
					i++
					// Check for escaped quote ('')
					if i < n && sql[i] == '\'' {
						current.WriteByte(sql[i])
						i++
						continue
					}
					break
				}
				// Handle backslash escape (MySQL style)
				if sql[i] == '\\' && i+1 < n {
					current.WriteByte(sql[i])
					i++
					current.WriteByte(sql[i])
					i++
					continue
				}
				current.WriteByte(sql[i])
				i++
			}
			continue
		}

		// Check for double-quoted string/identifier
		if ch == '"' {
			current.WriteByte(ch)
			i++
			for i < n {
				if sql[i] == '"' {
					current.WriteByte(sql[i])
					i++
					// Check for escaped quote ("")
					if i < n && sql[i] == '"' {
						current.WriteByte(sql[i])
						i++
						continue
					}
					break
				}
				current.WriteByte(sql[i])
				i++
			}
			continue
		}

		// Check for statement terminator
		if ch == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			i++
			continue
		}

		// Regular character
		current.WriteByte(ch)
		i++
	}

	// Don't forget the last statement (may not end with semicolon)
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// IsSelectStatement returns true if the statement appears to be a SELECT query
// (or other query that returns rows like SHOW, DESCRIBE, EXPLAIN, etc.)
func IsSelectStatement(stmt string) bool {
	// Trim leading whitespace and get the first word
	trimmed := strings.TrimLeftFunc(stmt, unicode.IsSpace)

	// Handle common prefixes like WITH (CTE)
	keywords := []string{
		"SELECT",
		"WITH",             // CTE that typically ends in SELECT
		"SHOW",             // MySQL SHOW commands
		"DESCRIBE", "DESC", // MySQL DESCRIBE
		"EXPLAIN", // Query plan
		"TABLE",   // PostgreSQL TABLE command (shorthand for SELECT * FROM)
		"VALUES",  // VALUES expression
		"PRAGMA",  // SQLite PRAGMA (some return rows)
	}

	upper := strings.ToUpper(trimmed)
	for _, kw := range keywords {
		if strings.HasPrefix(upper, kw) {
			// Check it's followed by whitespace or end of string
			if len(upper) == len(kw) || !unicode.IsLetter(rune(upper[len(kw)])) {
				return true
			}
		}
	}

	return false
}
