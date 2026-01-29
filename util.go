package main

import "strings"

// truncateString truncates a string to maxLen, adding ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// padRight pads a string with spaces to reach the specified length
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// quoteIdentifier returns the identifier quote character for the database type
func quoteIdentifier(dbType string) string {
	switch dbType {
	case "mysql":
		return "`"
	case "postgres", "postgresql", "pg":
		return `"`
	case "sqlite", "sqlite3":
		return `"`
	default:
		return `"`
	}
}
