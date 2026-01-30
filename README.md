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
- **Saved Connections**: Store database connections securely with encryption
- **Themes**: Visual themes to distinguish between environments (e.g., red for production)

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
| `-dsn` | Database connection string (use this OR `-conn`) |
| `-conn` | Named connection from `~/.dibber.yaml` |
| `-type` | Database type: `mysql`, `postgres`, `sqlite` (auto-detected from DSN) |
| `-sql-file` | SQL file to sync with query editor (default: `dibber.sql`) |
| `-format` | Output format for pipe mode: `table`, `csv`, `tsv` (default: `table`) |

### Connection Management Options

| Option | Description |
|--------|-------------|
| `-add-conn` | Add a new named connection (requires `-dsn`) |
| `-remove-conn` | Remove a saved connection |
| `-list-conns` | List all saved connections |
| `-change-password` | Change the master password |
| `-theme` | Theme for the connection (use with `-add-conn`) |
| `-list-themes` | List all available themes |

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

### Adding a Connection

```bash
# Add a connection with a name
dibber -add-conn mydb -dsn 'postgres://user:pass@localhost/mydb'

# Add with a specific theme
dibber -add-conn prod -dsn 'postgres://...' -theme production

# Add with explicit database type
dibber -add-conn legacy -dsn '...' -type mysql -theme gruvbox
```

On first use, you'll be prompted to create a master password. This password protects all your saved connections.

### Using a Saved Connection

```bash
# Interactive mode
dibber -conn mydb

# Pipe mode
echo 'SELECT * FROM users' | dibber -conn mydb -format csv
```

You'll be prompted for your master password to unlock the connection vault.

### Managing Connections

```bash
# List all saved connections
dibber -list-conns

# Remove a connection
dibber -remove-conn mydb

# Change the master password
dibber -change-password
```

### Switching Connections at Runtime

Press **Ctrl+P** while running dibber to open the connection picker. If your vault is locked, you'll be prompted for your master password. Select a connection and press Enter to switch.

### Security & Encryption

Saved connections are protected with industry-standard encryption:

| Component | Implementation |
|-----------|----------------|
| **Key Derivation** | Argon2id (OWASP recommended) |
| **Encryption** | AES-256-GCM (authenticated encryption) |
| **Architecture** | Envelope encryption pattern |

**How it works:**

1. **Master Password** - You choose a master password (min 8 characters)
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
- The master password is never stored - only a derived key can decrypt the data key
- Each DSN uses a unique nonce, so identical DSNs produce different ciphertext
- File permissions are set to `0600` (owner read/write only)
- The data key is held in memory only while the vault is unlocked
- Changing your password re-encrypts the data key (DSNs don't need re-encryption)

**What's NOT protected:**

- Connection names and themes are stored in plaintext (only DSNs are encrypted)
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

## License

BSD 3-Clause License. See [LICENSE](LICENSE) for details.
