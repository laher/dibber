# Dibber

**Dibber is a terminal-based SQL client with data editing capabilities.**

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), dibber provides an interactive TUI for exploring and modifying data across MySQL, PostgreSQL, and SQLite databases. It also supports a non-interactive pipe mode for scripting and automation.

## Features

- **Interactive SQL Editor**: Write and execute SQL queries with a full-featured text area
- **Results Table**: View query results in a paginated, navigable table
- **Row Detail View**: Inspect and edit individual rows
- **Smart Data Types**: Proper handling of NULLs, numeric types, booleans, and text
- **SQL Generation**: Automatically generate UPDATE, DELETE, and INSERT statements from your edits
- **Pipe Mode**: Execute queries from stdin and output results in table, CSV, or TSV format
- **Multi-Database**: Works with MySQL, PostgreSQL, and SQLite

## Installation

```bash
go install github.com/laher/dibber@latest
```

Or build from source:

```bash
git clone https://github.com/laher/dibber.git
cd dibber
go build -o dibber .
```

## Quick Start

```bash
# Interactive mode
dibber -dsn 'postgres://user:pass@localhost/mydb'

# Pipe mode - quick queries from the command line
echo 'SELECT * FROM users LIMIT 10' | dibber -dsn 'postgres://...'

# Export to CSV
echo 'SELECT * FROM orders' | dibber -dsn '...' -format csv > orders.csv
```

## Usage

### Interactive Mode

```bash
dibber -dsn 'connection_string' [-type mysql|postgres|sqlite] [-sql-file filename.sql]
```

The interactive TUI lets you write queries, navigate results, and edit data. Query content is automatically synced to a SQL file (default: `dibber.sql`).

### Pipe Mode

When stdin is piped, dibber runs in non-interactive mode:

```bash
# Table output (default)
echo 'SELECT id, name FROM users' | dibber -dsn '...'

# CSV output
cat complex_query.sql | dibber -dsn '...' -format csv

# TSV output
echo 'SELECT * FROM logs' | dibber -dsn '...' -format tsv > logs.tsv
```

Pipe mode outputs results to stdout and row counts to stderr, making it easy to chain with other tools:

```bash
echo 'SELECT * FROM users' | dibber -dsn '...' -format csv | grep 'active'
```

### Options

| Option | Description |
|--------|-------------|
| `-dsn` | Database connection string (required) |
| `-type` | Database type: `mysql`, `postgres`, `sqlite` (auto-detected from DSN) |
| `-sql-file` | SQL file to sync with query editor (default: `dibber.sql`) |
| `-format` | Output format for pipe mode: `table`, `csv`, `tsv` (default: `table`) |

### Connection Examples

**MySQL:**
```bash
dibber -dsn 'user:password@tcp(localhost:3306)/database'
```

**PostgreSQL:**
```bash
dibber -dsn 'postgres://user:password@localhost:5432/database'
```

**SQLite:**
```bash
dibber -dsn '/path/to/database.db'
dibber -dsn ':memory:'  # In-memory database
```

### DSN Formats

| Database | Format |
|----------|--------|
| MySQL | `user:password@tcp(host:port)/database` |
| PostgreSQL | `postgres://user:password@host:port/database` |
| SQLite | `/path/to/file.db` or `:memory:` |

## Key Bindings

### Query View

The query editor supports multiple queries separated by semicolons (`;`). When you execute, only the query under the cursor runs.

| Key | Action |
|-----|--------|
| `Ctrl+R` or `F5` | Execute query under cursor |
| `Tab` | Switch focus to results |
| `Ctrl+O` | Open file dialog |
| `Ctrl+Q` | Quit |

**Multi-query example:**
```sql
SELECT * FROM users;

SELECT * FROM orders
WHERE status = 'pending';

UPDATE users SET name = 'test' WHERE id = 1;
```

### Results View

| Key | Action |
|-----|--------|
| `↑` / `↓` or `j` / `k` | Navigate rows |
| `PgUp` / `PgDn` | Page navigation |
| `Ctrl+U` / `Ctrl+D` | Page up/down |
| `Home` / `End` or `g` / `G` | First/last row |
| `-` / `+` | Decrease/increase table height |
| `Enter` | Open detail view for selected row |
| `Tab` | Switch focus to query |
| `Esc` | Return to query view |

### Detail View

| Key | Action |
|-----|--------|
| `↑` / `↓` or `Tab` / `Shift+Tab` | Navigate fields |
| `PgUp` / `PgDn` | Scroll within multi-line content |
| `Ctrl+N` | Toggle NULL for current field |
| `Ctrl+U` or `F5` | Generate UPDATE statement |
| `Ctrl+D` or `F6` | Generate DELETE statement |
| `Ctrl+I` or `F7` | Generate INSERT statement |
| `Esc` | Return to results view |

## Data Editing

### Editability

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

### NULL Handling

- NULL values are visually distinguished from empty strings
- Press `Ctrl+N` to toggle a field between NULL and non-NULL
- Generated SQL correctly uses `NULL` keyword (not quoted strings)

### SQL Generation

From the detail view, you can generate SQL statements:

- **F5 (UPDATE)**: Generates an `UPDATE` with only changed fields
- **F6 (DELETE)**: Generates a `DELETE` for the current row
- **F7 (INSERT)**: Generates an `INSERT` with all field values (excluding auto-generated ID)

Generated statements are **appended** to the query editor. Press `Ctrl+R` to execute.

## Supported Databases

- **MySQL** - via [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)
- **PostgreSQL** - via [pgx](https://github.com/jackc/pgx)
- **SQLite** - via [go-sqlite3](https://github.com/mattn/go-sqlite3)

## Why "Dibber"?

<img src="https://upload.wikimedia.org/wikipedia/commons/4/45/Dibble_%28PSF%29.png" alt="A dibber" width="200" align="right">

A [dibber](https://en.wikipedia.org/wiki/Dibber) is a pointed wooden stick for making holes in the ground so that seeds, seedlings, or small bulbs can be planted. They come in a variety of designs including the straight dibber, T-handled dibber, trowel dibber, and L-shaped dibber.

Like its namesake, this tool helps you dig into your data and plant new rows.

It's also a somewhat childish soundalike for "d-b" (database).

## License

BSD 3-Clause License. See [LICENSE](LICENSE) for details.
