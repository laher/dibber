package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// executeQuery runs the SQL query and returns results
func executeQuery(db *sql.DB, query string) *QueryResult {
	rows, err := db.Query(query)
	if err != nil {
		return &QueryResult{Error: err}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return &QueryResult{Error: err}
	}

	var resultRows [][]string
	for rows.Next() {
		// Create a slice of interface{} to hold each column
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return &QueryResult{Error: err}
		}

		// Convert to strings
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
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{Error: err}
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
	}
}

// parseQueryMeta analyzes the query to determine if it's editable
func parseQueryMeta(query string, result *QueryResult) *QueryMeta {
	if result == nil || result.Error != nil {
		return nil
	}

	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	// Must be a SELECT query
	if !strings.HasPrefix(upperQuery, "SELECT") {
		return nil
	}

	// Check for aggregation functions that make it non-editable
	aggregateFuncs := []string{"COUNT(", "SUM(", "AVG(", "MIN(", "MAX(", "GROUP_CONCAT(", "GROUP BY", "HAVING", "DISTINCT"}
	for _, agg := range aggregateFuncs {
		if strings.Contains(upperQuery, agg) {
			return &QueryMeta{IsEditable: false}
		}
	}

	// Check for JOINs
	if strings.Contains(upperQuery, " JOIN ") {
		return &QueryMeta{IsEditable: false}
	}

	// Check for subqueries
	fromIdx := strings.Index(upperQuery, " FROM ")
	if fromIdx == -1 {
		return &QueryMeta{IsEditable: false}
	}

	// Look for multiple tables (comma in FROM clause before WHERE)
	afterFrom := query[fromIdx+6:]
	whereIdx := strings.Index(strings.ToUpper(afterFrom), " WHERE ")
	tablePart := afterFrom
	if whereIdx != -1 {
		tablePart = afterFrom[:whereIdx]
	}

	// Also check for ORDER BY, LIMIT etc
	for _, keyword := range []string{" ORDER BY ", " LIMIT ", " GROUP BY "} {
		if idx := strings.Index(strings.ToUpper(tablePart), keyword); idx != -1 {
			tablePart = tablePart[:idx]
		}
	}

	tablePart = strings.TrimSpace(tablePart)

	// Check for multiple tables
	if strings.Contains(tablePart, ",") {
		return &QueryMeta{IsEditable: false}
	}

	// Extract table name (handle backticks and aliases)
	tableName := extractTableName(tablePart)
	if tableName == "" {
		return &QueryMeta{IsEditable: false}
	}

	// Check if result has an 'id' column
	idIndex := -1
	idColumn := ""
	for i, col := range result.Columns {
		colLower := strings.ToLower(col)
		if colLower == "id" {
			idIndex = i
			idColumn = col
			break
		}
	}

	if idIndex == -1 {
		return &QueryMeta{IsEditable: false}
	}

	return &QueryMeta{
		TableName:  tableName,
		IsEditable: true,
		IDColumn:   idColumn,
		IDIndex:    idIndex,
	}
}

// extractTableName extracts the table name from a FROM clause fragment
func extractTableName(tablePart string) string {
	tablePart = strings.TrimSpace(tablePart)

	// Remove backticks
	tablePart = strings.ReplaceAll(tablePart, "`", "")

	// Handle alias (e.g., "users u" or "users AS u")
	parts := strings.Fields(tablePart)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

// getQueryUnderCursor finds and returns the SQL query that contains the cursor position
func (m Model) getQueryUnderCursor() string {
	content := m.textarea.Value()
	if strings.TrimSpace(content) == "" {
		return ""
	}

	// Get cursor line (0-indexed)
	cursorLine := m.textarea.Line()

	// Split content into lines and find which query block the cursor is in
	lines := strings.Split(content, "\n")

	// Calculate the character position at the start of the cursor line
	cursorPos := 0
	for i := 0; i < cursorLine && i < len(lines); i++ {
		cursorPos += len(lines[i]) + 1 // +1 for newline
	}
	// Add some offset into the current line (middle of line is fine for finding the query)
	if cursorLine < len(lines) {
		cursorPos += len(lines[cursorLine]) / 2
	}

	// Find all semicolon positions
	var semicolonPositions []int
	for i, ch := range content {
		if ch == ';' {
			semicolonPositions = append(semicolonPositions, i)
		}
	}

	// If no semicolons, there are no complete queries
	if len(semicolonPositions) == 0 {
		return ""
	}

	// Find which query segment contains the cursor
	// Query segments are: [0, semi1], [semi1+1, semi2], [semi2+1, semi3], ...
	queryStart := 0
	for _, semiPos := range semicolonPositions {
		if cursorPos <= semiPos {
			// Cursor is within this query (from queryStart to semiPos)
			query := strings.TrimSpace(content[queryStart : semiPos+1])
			// Remove the trailing semicolon for execution
			query = strings.TrimSuffix(query, ";")
			query = strings.TrimSpace(query)
			return query
		}
		queryStart = semiPos + 1
	}

	// Cursor is after the last semicolon - check if there's an incomplete query
	// If so, return empty (no complete query under cursor)
	remaining := strings.TrimSpace(content[queryStart:])
	if remaining == "" {
		// Cursor is right after last semicolon, return the last query
		if len(semicolonPositions) > 0 {
			lastSemi := semicolonPositions[len(semicolonPositions)-1]
			prevStart := 0
			if len(semicolonPositions) > 1 {
				prevStart = semicolonPositions[len(semicolonPositions)-2] + 1
			}
			query := strings.TrimSpace(content[prevStart : lastSemi+1])
			query = strings.TrimSuffix(query, ";")
			query = strings.TrimSpace(query)
			return query
		}
	}

	// There's incomplete text after last semicolon - no complete query under cursor
	return ""
}

// generateUpdateSQL creates an UPDATE statement from the edited fields
func (m Model) generateUpdateSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	// Get quote character based on database type
	q := quoteIdentifier(m.dbType)

	var setClauses []string
	for i, input := range m.detailView.inputs {
		newVal := input.Value()
		oldVal := m.detailView.originalRow[i]
		if newVal != oldVal {
			colName := m.result.Columns[i]
			// Escape single quotes
			escapedVal := strings.ReplaceAll(newVal, "'", "''")
			if newVal == "NULL" {
				setClauses = append(setClauses, fmt.Sprintf("%s%s%s = NULL", q, colName, q))
			} else {
				setClauses = append(setClauses, fmt.Sprintf("%s%s%s = '%s'", q, colName, q, escapedVal))
			}
		}
	}

	if len(setClauses) == 0 {
		return ""
	}

	// Get the ID value
	idVal := m.detailView.originalRow[m.queryMeta.IDIndex]
	escapedID := strings.ReplaceAll(idVal, "'", "''")

	return fmt.Sprintf("UPDATE %s%s%s SET %s WHERE %s%s%s = '%s'",
		q, m.queryMeta.TableName, q,
		strings.Join(setClauses, ", "),
		q, m.queryMeta.IDColumn, q,
		escapedID)
}

// generateDeleteSQL creates a DELETE statement for the current row
func (m Model) generateDeleteSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	q := quoteIdentifier(m.dbType)

	// Get the ID value
	idVal := m.detailView.originalRow[m.queryMeta.IDIndex]
	escapedID := strings.ReplaceAll(idVal, "'", "''")

	return fmt.Sprintf("DELETE FROM %s%s%s WHERE %s%s%s = '%s'",
		q, m.queryMeta.TableName, q,
		q, m.queryMeta.IDColumn, q,
		escapedID)
}

// generateInsertSQL creates an INSERT statement from the current field values
func (m Model) generateInsertSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	q := quoteIdentifier(m.dbType)

	var columns []string
	var values []string

	for i, input := range m.detailView.inputs {
		colName := m.result.Columns[i]
		val := input.Value()

		// Skip the ID column for INSERT (let the database auto-generate it)
		if i == m.queryMeta.IDIndex {
			continue
		}

		columns = append(columns, fmt.Sprintf("%s%s%s", q, colName, q))

		if val == "NULL" {
			values = append(values, "NULL")
		} else {
			escapedVal := strings.ReplaceAll(val, "'", "''")
			values = append(values, fmt.Sprintf("'%s'", escapedVal))
		}
	}

	return fmt.Sprintf("INSERT INTO %s%s%s (%s) VALUES (%s)",
		q, m.queryMeta.TableName, q,
		strings.Join(columns, ", "),
		strings.Join(values, ", "))
}
