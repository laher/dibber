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
dabble -dsn 'connection_string' [-type mysql|postgres|sqlite] [-sql-file filename.sql]
```

**Options:**
- `-type` - Database type (auto-detected from DSN if not specified)
- `-sql-file` - SQL file to sync with the query window (default: `dabble.sql`)

The query window content is automatically loaded from and saved to the SQL file. Saves occur when:
- Executing a query
- Generating UPDATE/DELETE/INSERT statements

### Connection Examples

**MySQL:**
```bash
dabble -dsn 'user:password@tcp(localhost:3306)/database'
```

**PostgreSQL:**
```bash
dabble -dsn 'postgres://user:password@localhost:5432/database'
```

**SQLite:**
```bash
dabble -dsn '/path/to/database.db'
dabble -dsn ':memory:'  # In-memory database
```

### DSN Formats

| Database | Format |
|----------|--------|
| MySQL | `user:password@tcp(host:port)/database` |
| PostgreSQL | `postgres://user:password@host:port/database` |
| SQLite | `/path/to/file.db` or `:memory:` |

## Key Bindings

### Query View

The query editor supports multiple queries separated by semicolons (`;`). When you execute, only the query under the cursor is run.

| Key | Action |
|-----|--------|
| `Ctrl+Enter` / `F5` | Execute query under cursor |
| `Tab` | Switch focus to results |
| `↑` / `↓` | Scroll through queries |
| `Esc` | Quit |

**Multi-query example:**
```sql
SELECT * FROM users;

SELECT * FROM orders
WHERE status = 'pending';

UPDATE users SET name = 'test' WHERE id = 1;
```
Position your cursor on any query and press `Ctrl+Enter` to execute just that query.

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
| `F6` | Generate DELETE statement (if editable) |
| `F7` | Generate INSERT statement (if editable) |
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

When you're in the detail view, you can generate SQL statements:

- **F5 (UPDATE)**: Generates an `UPDATE` statement with only the changed fields
- **F6 (DELETE)**: Generates a `DELETE` statement for the current row
- **F7 (INSERT)**: Generates an `INSERT` statement with all field values (excluding the ID, which is auto-generated)

All generated statements are **appended** to the query editor (not replacing existing content) and the cursor moves to the new query. Press `Ctrl+Enter` to execute.

## Supported Databases

- **MySQL** - via [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
- **PostgreSQL** - via [pgx](https://github.com/jackc/pgx)
- **SQLite** - via [go-sqlite3](https://github.com/mattn/go-sqlite3)

## License

BSD 3-Clause License. See [LICENSE](LICENSE) for details.
