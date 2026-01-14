# Dabble

A terminal-based database client written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Interactive SQL Query Editor**: Write and execute SQL queries with a full-featured text area
- **Results Table**: View query results in a paginated, navigable table
- **Row Detail View**: Inspect individual rows with Enter key
- **Inline Editing**: Edit data directly for simple queries (single table with `id` column)
- **UPDATE Statement Generation**: Automatically generate UPDATE statements from your edits

## Installation

```bash
go install github.com/laher/dabble@latest
```

Or build from source:

```bash
git clone https://github.com/laher/dabble.git
cd dabble
go build -o dabble .
```

## Usage

```bash
dabble -dsn 'user:password@tcp(localhost:3306)/database'
```

### DSN Format (MySQL)

```
user:password@tcp(host:port)/database
```

## Key Bindings

### Query View

| Key | Action |
|-----|--------|
| `Ctrl+Enter` / `F5` | Execute query |
| `Tab` | Switch focus to results |
| `Esc` | Quit |

### Results View

| Key | Action |
|-----|--------|
| `↑` / `↓` or `j` / `k` | Navigate rows |
| `PgUp` / `PgDn` | Page navigation |
| `Ctrl+U` / `Ctrl+D` | Page up/down |
| `Home` / `End` or `g` / `G` | First/last row |
| `Enter` | Open detail view for selected row |
| `Tab` | Switch focus to query |
| `Esc` | Return to query view |

### Detail View

| Key | Action |
|-----|--------|
| `↑` / `↓` or `Tab` / `Shift+Tab` | Navigate fields |
| `F5` | Generate UPDATE statement (if editable) |
| `Esc` | Return to results view |

## Editability

The detail view allows editing only when the query is "simple enough":

**Editable queries must:**

- Be a `SELECT` statement
- Query a single table (no JOINs)
- Return an `id` column

**Non-editable queries include:**

- Queries with `JOIN`s
- Queries with aggregation (`COUNT`, `SUM`, `AVG`, `MIN`, `MAX`, etc.)
- Queries with `GROUP BY`, `HAVING`, or `DISTINCT`
- Queries selecting from multiple tables

When you edit fields and press `F5`, Dabble generates an `UPDATE` statement and places it in the query editor. You can review and execute it manually.

## Supported Databases

- MySQL (current)

Additional database support planned for future releases.

## License

BSD 3-Clause License. See [LICENSE](LICENSE) for details.
