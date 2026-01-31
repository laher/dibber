package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SQLHighlighter provides SQL syntax highlighting for the query window
type SQLHighlighter struct {
	theme Theme

	// Compiled patterns
	keywordPattern  *regexp.Regexp
	functionPattern *regexp.Regexp
	stringPattern   *regexp.Regexp
	numberPattern   *regexp.Regexp
	commentPattern  *regexp.Regexp
	operatorPattern *regexp.Regexp
}

// NewSQLHighlighter creates a new SQL highlighter with the given theme
func NewSQLHighlighter(theme Theme) *SQLHighlighter {
	// SQL keywords (case-insensitive matching)
	keywords := []string{
		// DML
		"SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN", "LIKE", "BETWEEN",
		"IS", "NULL", "AS", "ON", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
		"CROSS", "FULL", "NATURAL", "USING", "ORDER", "BY", "ASC", "DESC",
		"LIMIT", "OFFSET", "GROUP", "HAVING", "DISTINCT", "ALL", "UNION",
		"INTERSECT", "EXCEPT", "INTO", "VALUES", "SET", "UPDATE", "DELETE",
		"INSERT", "REPLACE", "TRUNCATE", "CREATE", "ALTER", "DROP", "TABLE",
		"INDEX", "VIEW", "DATABASE", "SCHEMA", "IF", "EXISTS", "CASCADE",
		"CONSTRAINT", "PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE",
		"CHECK", "DEFAULT", "AUTO_INCREMENT", "AUTOINCREMENT",
		// Data types
		"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "FLOAT", "DOUBLE",
		"DECIMAL", "NUMERIC", "REAL", "BOOLEAN", "BOOL", "CHAR", "VARCHAR",
		"TEXT", "BLOB", "DATE", "TIME", "DATETIME", "TIMESTAMP", "SERIAL",
		// Transaction
		"BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION", "SAVEPOINT",
		// Other
		"CASE", "WHEN", "THEN", "ELSE", "END", "CAST", "CONVERT", "COALESCE",
		"NULLIF", "TRUE", "FALSE", "WITH", "RECURSIVE", "EXPLAIN", "ANALYZE",
	}

	// SQL aggregate and common functions
	functions := []string{
		"COUNT", "SUM", "AVG", "MIN", "MAX", "ROUND", "FLOOR", "CEIL", "ABS",
		"UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM", "LENGTH", "SUBSTR", "SUBSTRING",
		"REPLACE", "CONCAT", "CONCAT_WS", "COALESCE", "IFNULL", "NULLIF", "IIF",
		"NOW", "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "DATE",
		"YEAR", "MONTH", "DAY", "HOUR", "MINUTE", "SECOND", "STRFTIME",
		"PRINTF", "TYPEOF", "INSTR", "GROUP_CONCAT", "RANDOM", "HEX", "QUOTE",
	}

	// Build keyword pattern (word boundaries, case-insensitive)
	keywordStr := `(?i)\b(` + strings.Join(keywords, "|") + `)\b`

	// Build function pattern (word followed by open paren)
	funcStr := `(?i)\b(` + strings.Join(functions, "|") + `)\s*\(`

	return &SQLHighlighter{
		theme:           theme,
		keywordPattern:  regexp.MustCompile(keywordStr),
		functionPattern: regexp.MustCompile(funcStr),
		stringPattern:   regexp.MustCompile(`'[^']*'|"[^"]*"`),
		numberPattern:   regexp.MustCompile(`\b-?\d+\.?\d*\b`),
		commentPattern:  regexp.MustCompile(`--.*$|/\*[\s\S]*?\*/`),
		operatorPattern: regexp.MustCompile(`[<>=!]+|[+\-*/%]|\|\||&&`),
	}
}

// tokenType represents the type of SQL token
type tokenType int

const (
	tokenText tokenType = iota
	tokenKeyword
	tokenFunction
	tokenString
	tokenNumber
	tokenComment
	tokenOperator
)

// token represents a highlighted token
type token struct {
	text  string
	typ   tokenType
	start int
	end   int
}

// Highlight applies syntax highlighting to SQL text and returns styled string
func (h *SQLHighlighter) Highlight(sql string) string {
	if sql == "" {
		return ""
	}

	// Tokenize the SQL
	tokens := h.tokenize(sql)

	// Build the highlighted output
	var result strings.Builder
	lastEnd := 0

	for _, tok := range tokens {
		// Add any text before this token (unhighlighted)
		if tok.start > lastEnd {
			result.WriteString(sql[lastEnd:tok.start])
		}

		// Add the highlighted token
		result.WriteString(h.styleToken(tok))
		lastEnd = tok.end
	}

	// Add any remaining text
	if lastEnd < len(sql) {
		result.WriteString(sql[lastEnd:])
	}

	return result.String()
}

// tokenize breaks SQL into tokens for highlighting
func (h *SQLHighlighter) tokenize(sql string) []token {
	var tokens []token
	covered := make([]bool, len(sql))

	// Find comments first (highest priority - they can contain anything)
	for _, match := range h.commentPattern.FindAllStringIndex(sql, -1) {
		if !h.isOverlapping(covered, match[0], match[1]) {
			tokens = append(tokens, token{
				text:  sql[match[0]:match[1]],
				typ:   tokenComment,
				start: match[0],
				end:   match[1],
			})
			h.markCovered(covered, match[0], match[1])
		}
	}

	// Find strings (second priority)
	for _, match := range h.stringPattern.FindAllStringIndex(sql, -1) {
		if !h.isOverlapping(covered, match[0], match[1]) {
			tokens = append(tokens, token{
				text:  sql[match[0]:match[1]],
				typ:   tokenString,
				start: match[0],
				end:   match[1],
			})
			h.markCovered(covered, match[0], match[1])
		}
	}

	// Find functions (before keywords, so COUNT( matches as function not keyword)
	for _, match := range h.functionPattern.FindAllStringSubmatchIndex(sql, -1) {
		// match[2]:match[3] is the function name (capture group 1)
		if match[2] >= 0 && !h.isOverlapping(covered, match[2], match[3]) {
			tokens = append(tokens, token{
				text:  sql[match[2]:match[3]],
				typ:   tokenFunction,
				start: match[2],
				end:   match[3],
			})
			h.markCovered(covered, match[2], match[3])
		}
	}

	// Find keywords
	for _, match := range h.keywordPattern.FindAllStringIndex(sql, -1) {
		if !h.isOverlapping(covered, match[0], match[1]) {
			tokens = append(tokens, token{
				text:  sql[match[0]:match[1]],
				typ:   tokenKeyword,
				start: match[0],
				end:   match[1],
			})
			h.markCovered(covered, match[0], match[1])
		}
	}

	// Find numbers
	for _, match := range h.numberPattern.FindAllStringIndex(sql, -1) {
		if !h.isOverlapping(covered, match[0], match[1]) {
			tokens = append(tokens, token{
				text:  sql[match[0]:match[1]],
				typ:   tokenNumber,
				start: match[0],
				end:   match[1],
			})
			h.markCovered(covered, match[0], match[1])
		}
	}

	// Find operators
	for _, match := range h.operatorPattern.FindAllStringIndex(sql, -1) {
		if !h.isOverlapping(covered, match[0], match[1]) {
			tokens = append(tokens, token{
				text:  sql[match[0]:match[1]],
				typ:   tokenOperator,
				start: match[0],
				end:   match[1],
			})
			h.markCovered(covered, match[0], match[1])
		}
	}

	// Sort tokens by start position
	for i := 0; i < len(tokens); i++ {
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].start < tokens[i].start {
				tokens[i], tokens[j] = tokens[j], tokens[i]
			}
		}
	}

	return tokens
}

func (h *SQLHighlighter) isOverlapping(covered []bool, start, end int) bool {
	for i := start; i < end && i < len(covered); i++ {
		if covered[i] {
			return true
		}
	}
	return false
}

func (h *SQLHighlighter) markCovered(covered []bool, start, end int) {
	for i := start; i < end && i < len(covered); i++ {
		covered[i] = true
	}
}

// styleToken applies the appropriate lipgloss style to a token
func (h *SQLHighlighter) styleToken(tok token) string {
	var style lipgloss.Style

	switch tok.typ {
	case tokenKeyword:
		style = lipgloss.NewStyle().
			Foreground(h.theme.SyntaxKeyword).
			Bold(true)
	case tokenFunction:
		style = lipgloss.NewStyle().
			Foreground(h.theme.SyntaxFunction)
	case tokenString:
		style = lipgloss.NewStyle().
			Foreground(h.theme.SyntaxString)
	case tokenNumber:
		style = lipgloss.NewStyle().
			Foreground(h.theme.SyntaxNumber)
	case tokenComment:
		style = lipgloss.NewStyle().
			Foreground(h.theme.SyntaxComment).
			Italic(true)
	case tokenOperator:
		style = lipgloss.NewStyle().
			Foreground(h.theme.SyntaxOperator)
	default:
		return tok.text
	}

	return style.Render(tok.text)
}

// HighlightLine highlights a single line of SQL
func (h *SQLHighlighter) HighlightLine(line string) string {
	return h.Highlight(line)
}
