# Dibber

<img src="https://upload.wikimedia.org/wikipedia/commons/4/45/Dibble_%28PSF%29.png" alt="A dibber" width="200" align="right">

[![Tests](https://github.com/laher/dibber/actions/workflows/test.yml/badge.svg)](https://github.com/laher/dibber/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/laher/dibber)](https://goreportcard.com/report/github.com/laher/dibber)
[![Go Reference](https://pkg.go.dev/badge/github.com/laher/dibber.svg)](https://pkg.go.dev/github.com/laher/dibber)
[![License](https://img.shields.io/github/license/laher/dibber)](LICENSE)

**Dibber is a terminal-based SQL client with data editing capabilities.**

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), dibber provides an interactive TUI for exploring and modifying data across MySQL, PostgreSQL, and SQLite databases. It also supports a non-interactive pipe mode for scripting and automation.

## Features

- **Interactive SQL Editor**: Write and execute SQL queries with a full-featured text area
- **Results Table**: View query results in a paginated, navigable table
- **Saved Connections**: Store database connections **securely with encryption**
- **Row Detail View**: Inspect and **edit** individual rows
- **SQL Generation**: Automatically generate UPDATE, DELETE, and INSERT statements from your edits
- **Pipe Mode**: Execute queries from stdin and output results in table, CSV, or TSV format
- **Multi-Database**: Works with MySQL, PostgreSQL, and SQLite
- **Themes**: Visual themes to distinguish between environments (e.g. red for production)

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
# Interactive mode with direct DSN
dibber -dsn 'postgres://user:pass@localhost/mydb'

# Save a connection for reuse (encrypted)
dibber -add-conn prod -dsn 'postgres://user:pass@prod-host/db' -theme production

# Use a saved connection
dibber -conn prod

# Pipe mode - quick queries from the command line
echo 'SELECT * FROM users LIMIT 10' | dibber -conn prod

# Export to CSV
echo 'SELECT * FROM orders' | dibber -conn prod -format csv > orders.csv
```

## Usage

### Interactive Mode

```bash
dibber -dsn 'connection_string' [-type mysql|postgres|sqlite] [-sql-file filename.sql]
```

The interactive TUI lets you write queries, navigate results, and edit data. Query content is automatically synced to a SQL file in the configured SQL directory (default: `$HOME/sql/dibber.sql`).

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
| `-dsn` | Database connection string (use this OR `-conn`) |
| `-conn` | Named connection from `~/.dibber.yaml` |
| `-type` | Database type: `mysql`, `postgres`, `sqlite` (auto-detected from DSN) |
| `-sql-dir` | Directory for SQL files (overrides config setting) |
| `-set-sql-dir` | Set the SQL directory in `~/.dibber.yaml` |
| `-sql-file` | SQL file to sync with query editor (default: `dibber.sql`, relative to sql-dir) |
| `-format` | Output format for pipe mode: `table`, `csv`, `tsv` (default: `table`) |

### Connection Management Options

| Option | Description |
|--------|-------------|
| `-add-conn` | Add a new named connection (requires `-dsn`) |
| `-remove-conn` | Remove a saved connection |
| `-list-conns` | List all saved connections |
| `-change-password` | Change the encryption password |
| `-theme` | Theme for the connection (use with `-add-conn`) |
| `-list-themes` | List all available themes |

### SQL Directory

SQL files are stored in a configurable directory. The default is `$HOME/sql`.

```bash
# Set the SQL directory in config (persisted to ~/.dibber.yaml)
dibber -set-sql-dir ~/my-sql-scripts

# Override for a single session
dibber -conn mydb -sql-dir /tmp/scratch
```

The SQL directory is created automatically if it doesn't exist. The file dialog (Ctrl+O) opens in this directory by default.

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

## Saved Connections

Dibber can store database connections for reuse. Connections are encrypted and stored in `~/.dibber.yaml`.

### Adding Connections via the UI (Recommended)

The most secure way to add connections is through the UI, as the DSN is never visible in shell history or process lists:

1. Start dibber with any connection (or even a SQLite memory database): `dibber -dsn ':memory:'`
2. Press **Ctrl+P** to open the Connection Manager
3. If this is your first time, you'll be prompted to create an encryption password
4. Press **a** to add a new connection
5. Enter a name for the connection (e.g., "prod", "dev", "staging")
6. Enter the DSN (displayed as dots for security)
7. Select the database type (auto-detected if possible)
8. Choose a theme (optional, but useful for distinguishing environments)
9. Press Enter to save

Your connection is now encrypted and stored securely.

### Adding Connections via Command Line

You can also add connections from the command line, though this is less secure as the DSN appears in shell history:

```bash
# Add a connection with a name
dibber -add-conn mydb -dsn 'postgres://user:pass@localhost/mydb'

# Add with a specific theme
dibber -add-conn prod -dsn 'postgres://...' -theme production

# Add with explicit database type
dibber -add-conn legacy -dsn '...' -type mysql -theme gruvbox
```

On first use, you'll be prompted to create an encryption password. This password protects all your saved connections.

### Using a Saved Connection

```bash
# Interactive mode
dibber -conn mydb

# Pipe mode
echo 'SELECT * FROM users' | dibber -conn mydb -format csv
```

You'll be prompted for your encryption password to unlock the connection vault.

### Managing Connections

**Via UI (Ctrl+P):**

- Press **a** to add a new connection
- Press **d** to delete the selected connection
- Use **↑↓** to navigate, **Enter** to connect

**Via Command Line:**

```bash
# List all saved connections
dibber -list-conns

# Remove a connection
dibber -remove-conn mydb

# Change the encryption password
dibber -change-password
```

### Connection Manager (Ctrl+P)

Press **Ctrl+P** at any time to open the Connection Manager. This provides a complete interface for managing your saved connections:

| Key | Action |
|-----|--------|
| `↑↓` | Navigate connections |
| `Enter` | Connect to selected |
| `a` or `n` | Add new connection |
| `d` or `x` | Delete selected connection |
| `Esc` | Close manager |

If the vault is locked, you'll be prompted for your encryption password first. If no vault exists, you'll be guided through creating one.

### Security & Encryption

Saved connections are protected with industry-standard encryption:

| Component | Implementation |
|-----------|----------------|
| **Key Derivation** | Argon2id (OWASP recommended) |
| **Encryption** | AES-256-GCM (authenticated encryption) |
| **Architecture** | Envelope encryption pattern |

**How it works:**

1. **encryption Password** - You choose a encryption password (min 8 characters)
2. **Key Derivation** - Argon2id derives a key from your password + random salt
   - Parameters: 64MB memory, 3 iterations, 4 threads
   - This makes brute-force attacks computationally expensive
3. **Data Key** - A random 256-bit data key is generated and encrypted with the derived key
4. **DSN Encryption** - Each DSN is encrypted with AES-256-GCM using the data key
   - Each encryption uses a unique random nonce
   - GCM provides authentication (tamper detection)
5. **Storage** - `~/.dibber.yaml` stores:
   - Salt (for key derivation)
   - Encrypted data key
   - Encrypted DSNs (with nonce prepended)

**Security properties:**

- DSNs are never stored in plaintext
- The encryption password is never stored - only a derived key can decrypt the data key
- Each DSN uses a unique nonce, so identical DSNs produce different ciphertext
- File permissions are set to `0600` (owner read/write only)
- The data key is held in memory only while the vault is unlocked
- Changing your password re-encrypts the data key (DSNs don't need re-encryption)
- **UI-based entry (Ctrl+P) keeps DSNs out of shell history and process lists**

**What's NOT protected:**

- Connection names and themes are stored in plaintext (only DSNs are encrypted)
- Command-line DSN entry (`-add-conn -dsn '...'`) appears in shell history
- Memory is not securely wiped (Go doesn't guarantee secure memory erasure)
- No protection against keyloggers or malware with memory access

## Themes

Themes change the color scheme of the UI, making it easy to visually distinguish between environments.

### Available Themes

| Theme | Description |
|-------|-------------|
| `default` | Default purple theme |
| `dracula` | Dracula dark theme |
| `monokai` | Classic Monokai theme |
| `nord` | Arctic Nord theme |
| `gruvbox` | Retro Gruvbox theme |
| `tokyo-night` | Tokyo Night theme |
| `catppuccin` | Catppuccin Mocha theme |
| `solarized` | Solarized Dark theme |
| `forest` | Calming green forest theme |
| `ocean` | Deep ocean blue theme |
| `production` | **Red warning theme for production databases** |

```bash
# List all available themes
dibber -list-themes
```

### Using Themes

Themes are associated with saved connections:

```bash
# Add a connection with a theme
dibber -add-conn prod -dsn '...' -theme production
dibber -add-conn dev -dsn '...' -theme dracula
dibber -add-conn staging -dsn '...' -theme nord

# The theme applies automatically when you use the connection
dibber -conn prod  # Red UI - unmistakably production!
```

The title bar shows the current theme when using a non-default theme:

```
🌱  Dibber - prod (postgres) [production]
```

### The Production Theme

The `production` theme uses aggressive red coloring throughout the UI. This makes it immediately obvious when you're connected to a production database, reducing the risk of accidentally running destructive queries in the wrong environment.

## Key Bindings

### Global Keys

| Key | Action |
|-----|--------|
| `Ctrl+P` | Open connection picker (switch databases) |
| `Ctrl+S` | Save SQL file |
| `Ctrl+Q` | Quit |

### Query View

The query editor supports multiple queries separated by semicolons (`;`). When you execute, only the query under the cursor runs.

| Key | Action |
|-----|--------|
| `Ctrl+R` or `F5` | Execute query under cursor |
| `Tab` | Switch focus to results |
| `Ctrl+O` | Open file dialog |

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

## TODOs

- Make it optional to encrypt DSNs in config file. e.g. local databases often don't need it (and therefore, no need for entering encryption password)
- Tabs for multiple connections
- Refine the concept of 'modal editor' - Esc to go to results view, providing
 more key mappings.
- Maybe menus
- 'rollover' sql file to back up sql file and clear it
- Different default file per named connection? configurable (so that connections can share if needed)
- Improve cursor - multiline selection
- [not sure] autocompletion of SQL keywords and maybe table/column names
- refine 'pipe mode'
- pipe to external app for fetching encryption password? e.g. pass, op
(1password), etc

## License

BSD 3-Clause License. See [LICENSE](LICENSE) for details.
