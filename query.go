package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// executeQuery runs the SQL query and returns results with type information
func executeQuery(db *sql.DB, query string) *QueryResult {
	rows, err := db.Query(query)
	if err != nil {
		return &QueryResult{Error: err}
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return &QueryResult{Error: err}
	}

	// Get column type information
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return &QueryResult{Error: err}
	}

	// Map database types to our ColumnType categories
	colTypes := make([]ColumnType, len(columns))
	for i, ct := range columnTypes {
		colTypes[i] = categorizeColumnType(ct.DatabaseTypeName())
	}

	var resultRows [][]CellValue
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

		// Convert to CellValues with NULL awareness
		row := make([]CellValue, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = CellValue{Value: "", IsNull: true}
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = CellValue{Value: string(v), IsNull: false}
				case bool:
					if v {
						row[i] = CellValue{Value: "true", IsNull: false}
					} else {
						row[i] = CellValue{Value: "false", IsNull: false}
					}
				default:
					row[i] = CellValue{Value: fmt.Sprintf("%v", v), IsNull: false}
				}
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{Error: err}
	}

	return &QueryResult{
		Columns:     columns,
		ColumnTypes: colTypes,
		Rows:        resultRows,
	}
}

// categorizeColumnType maps database-specific type names to our general categories
func categorizeColumnType(dbTypeName string) ColumnType {
	typeName := strings.ToUpper(dbTypeName)

	// Numeric types
	numericTypes := []string{
		"INT", "INTEGER", "SMALLINT", "BIGINT", "TINYINT", "MEDIUMINT",
		"DECIMAL", "NUMERIC", "FLOAT", "DOUBLE", "REAL",
		"INT2", "INT4", "INT8", "FLOAT4", "FLOAT8",
		"SERIAL", "BIGSERIAL", "SMALLSERIAL",
		"MONEY",
	}
	for _, nt := range numericTypes {
		if strings.Contains(typeName, nt) {
			return ColTypeNumeric
		}
	}

	// Boolean types
	booleanTypes := []string{"BOOL", "BOOLEAN", "BIT"}
	for _, bt := range booleanTypes {
		if strings.Contains(typeName, bt) {
			return ColTypeBoolean
		}
	}

	// Date/time types
	dateTypes := []string{
		"DATE", "TIME", "DATETIME", "TIMESTAMP",
		"TIMESTAMPTZ", "TIMETZ",
		"YEAR", "INTERVAL",
	}
	for _, dt := range dateTypes {
		if strings.Contains(typeName, dt) {
			return ColTypeDatetime
		}
	}

	// Blob/binary types
	blobTypes := []string{
		"BLOB", "BINARY", "VARBINARY", "BYTEA",
		"TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
	}
	for _, bt := range blobTypes {
		if strings.Contains(typeName, bt) {
			return ColTypeBlob
		}
	}

	// Text types (default for most string-like types)
	textTypes := []string{
		"CHAR", "VARCHAR", "TEXT", "STRING",
		"TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"NCHAR", "NVARCHAR", "NTEXT",
		"UUID", "JSON", "JSONB", "XML",
		"ENUM", "SET",
	}
	for _, tt := range textTypes {
		if strings.Contains(typeName, tt) {
			return ColTypeText
		}
	}

	// Default to text for unknown types (safer to quote)
	return ColTypeUnknown
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

// formatValueForSQL formats a value for use in a SQL statement based on type and NULL state
func formatValueForSQL(value string, isNull bool, colType ColumnType, dbType string) string {
	if isNull {
		return "NULL"
	}

	// Handle empty string - it's a valid value, not NULL
	if value == "" {
		return "''"
	}

	// Numeric types don't need quotes
	if colType.IsNumeric() {
		// Validate it looks like a number to prevent SQL injection
		if isValidNumber(value) {
			return value
		}
		// If not a valid number, quote it (safer, will cause DB error if wrong)
		return fmt.Sprintf("'%s'", escapeSQLString(value))
	}

	// Boolean handling
	if colType.IsBoolean() {
		lower := strings.ToLower(value)
		switch lower {
		case "true", "1", "yes", "on", "t":
			switch dbType {
			case "mysql":
				return "1"
			default:
				return "TRUE"
			}
		case "false", "0", "no", "off", "f":
			switch dbType {
			case "mysql":
				return "0"
			default:
				return "FALSE"
			}
		}
		// Invalid boolean, quote it
		return fmt.Sprintf("'%s'", escapeSQLString(value))
	}

	// Text and other types get quoted
	return fmt.Sprintf("'%s'", escapeSQLString(value))
}

// escapeSQLString escapes single quotes in a string for SQL
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// isValidNumber checks if a string represents a valid SQL number
func isValidNumber(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Handle negative numbers
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	if s == "" {
		return false
	}

	hasDecimal := false
	hasDigit := false

	for i, ch := range s {
		if ch >= '0' && ch <= '9' {
			hasDigit = true
			continue
		}
		if ch == '.' && !hasDecimal {
			hasDecimal = true
			continue
		}
		// Allow scientific notation
		if (ch == 'e' || ch == 'E') && hasDigit && i < len(s)-1 {
			rest := s[i+1:]
			if rest[0] == '+' || rest[0] == '-' {
				rest = rest[1:]
			}
			// Rest must be all digits
			for _, r := range rest {
				if r < '0' || r > '9' {
					return false
				}
			}
			return len(rest) > 0
		}
		return false
	}

	return hasDigit
}

// generateUpdateSQL creates an UPDATE statement from the edited fields
func (m Model) generateUpdateSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	q := quoteIdentifier(m.dbType)

	var setClauses []string
	for i, input := range m.detailView.inputs {
		newVal := input.Value()
		newIsNull := m.detailView.isNull[i]
		origVal := m.detailView.originalValues[i]

		// Check if value has changed (compare both value and NULL state)
		valueChanged := newVal != origVal.Value || newIsNull != origVal.IsNull

		if valueChanged {
			colName := m.result.Columns[i]
			colType := m.detailView.columnTypes[i]
			formattedVal := formatValueForSQL(newVal, newIsNull, colType, m.dbType)
			setClauses = append(setClauses, fmt.Sprintf("%s%s%s = %s", q, colName, q, formattedVal))
		}
	}

	if len(setClauses) == 0 {
		return ""
	}

	// Get the ID value (use original, never NULL for WHERE clause)
	idVal := m.detailView.originalValues[m.queryMeta.IDIndex]
	idColType := m.detailView.columnTypes[m.queryMeta.IDIndex]
	formattedID := formatValueForSQL(idVal.Value, false, idColType, m.dbType)

	return fmt.Sprintf("UPDATE %s%s%s SET %s WHERE %s%s%s = %s",
		q, m.queryMeta.TableName, q,
		strings.Join(setClauses, ", "),
		q, m.queryMeta.IDColumn, q,
		formattedID)
}

// generateDeleteSQL creates a DELETE statement for the current row
func (m Model) generateDeleteSQL() string {
	if m.detailView == nil || m.queryMeta == nil || !m.queryMeta.IsEditable {
		return ""
	}

	q := quoteIdentifier(m.dbType)

	// Get the ID value
	idVal := m.detailView.originalValues[m.queryMeta.IDIndex]
	idColType := m.detailView.columnTypes[m.queryMeta.IDIndex]
	formattedID := formatValueForSQL(idVal.Value, false, idColType, m.dbType)

	return fmt.Sprintf("DELETE FROM %s%s%s WHERE %s%s%s = %s",
		q, m.queryMeta.TableName, q,
		q, m.queryMeta.IDColumn, q,
		formattedID)
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
		// Skip the ID column for INSERT (let the database auto-generate it)
		if i == m.queryMeta.IDIndex {
			continue
		}

		colName := m.result.Columns[i]
		val := input.Value()
		isNull := m.detailView.isNull[i]
		colType := m.detailView.columnTypes[i]

		columns = append(columns, fmt.Sprintf("%s%s%s", q, colName, q))
		values = append(values, formatValueForSQL(val, isNull, colType, m.dbType))
	}

	return fmt.Sprintf("INSERT INTO %s%s%s (%s) VALUES (%s)",
		q, m.queryMeta.TableName, q,
		strings.Join(columns, ", "),
		strings.Join(values, ", "))
}
