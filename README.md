# PostgreSQL Backup Utility

A lightweight, long-running backup service written in Go that automatically backs up a PostgreSQL database at regular intervals using `pg_dump`.

## Features

- **Configurable scheduled backups** — backup interval set via `config.yaml`, with an immediate backup on startup
- **Timestamped backup files** — each backup is saved as `finpay_YYYYMMDD_HHMMSS.sql`
- **Backup retention** — automatically keeps only the latest 3 backups and deletes older ones
- **Retry mechanism** — retries up to 3 times with a 10-second delay between attempts on failure
- **Backup validation** — verifies the backup file exists and is non-empty after `pg_dump` completes
- **Failed backup cleanup** — removes incomplete or empty backup files to avoid storing corrupt data

## Technologies

- **Go** with [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) for configuration parsing
- **PostgreSQL** `pg_dump` CLI tool

## Project Structure

```
postgres-backup/
├── main.go          # Application source code
├── config.yaml      # Backup interval configuration
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
├── .gitignore       # Ignores backup files, binaries, .env
├── backup/          # Backup output directory (created automatically)
│   └── finpay_*.sql # Timestamped backup files
└── README.md
```

## How It Works

1. On startup, the service reads `config.yaml` to determine the backup interval.
2. An immediate backup is triggered.
3. `pg_dump` is executed to dump the `finpay` database to a timestamped `.sql` file in the `backup/` directory.
4. After a successful dump, the service validates that the file exists and is non-empty.
5. If backup fails, incomplete/empty files are cleaned up and the operation is retried (up to 3 attempts).
6. After a successful backup, old backups beyond the retention limit (3) are deleted.
7. The service then waits for the configured interval before repeating the process.

## Configuration

Edit `config.yaml` in the project root to set the backup interval:

```yaml
backup_interval: 8h
```

The value uses Go's [`time.Duration`](https://pkg.go.dev/time#ParseDuration) format — valid examples: `30m`, `2h`, `4h30m`, `24h`. The interval can be changed without modifying or recompiling the Go source code — just update the file and restart the service.

> **Note:** `config.yaml` contains only scheduling settings. Database passwords are **not** stored here — see [PostgreSQL Authentication](#postgresql-authentication) below.

## PostgreSQL Authentication

The application connects to PostgreSQL with these parameters:

| Parameter | Value       |
|-----------|-------------|
| Host      | `localhost` |
| Port      | `5432`      |
| User      | `postgres`  |
| Database  | `finpay`    |

Password is handled by PostgreSQL's standard authentication mechanisms. To avoid interactive password prompts, configure one of the following:

- **`pgpass` file** — create `%APPDATA%\postgresql\pgpass.conf` (Windows) with the entry:
  ```
  localhost:5432:finpay:postgres:YOUR_PASSWORD
  ```
- **`PGPASSWORD` environment variable** — set before running the application:
  ```bash
  set PGPASSWORD=your_password
  ```

> **Note:** Never commit passwords to version control. The `.gitignore` already excludes `.env` files.

## Usage

### Prerequisites

- Go 1.26+ installed
- PostgreSQL installed with `pg_dump` available on the system PATH
- Access credentials for the target database

### Run directly

```bash
go run main.go
```

### Build and run (Windows)

```bash
go build -o postgres-backup.exe main.go
.\postgres-backup.exe
```

## Example Output

```
2026/09/16 11:34:27 PostgreSQL backup service started.
Backup interval: 8h0m0s
Backup attempt 1 of 3
Starting PostgreSQL backup...
Backup file: backup\finpay_20260916_113427.sql
Backup completed successfully!
Backup size: 20083 bytes
Backup process completed successfully.
```

## Verification

To confirm backups are working:

1. **Check the `backup/` directory** — it should contain up to 3 timestamped `.sql` files.
2. **Inspect a backup file** — open any `.sql` file to verify it contains valid SQL statements.
3. **Test a restore** (on a test database):
   ```bash
   psql -h localhost -p 5432 -U postgres -d test_db -f backup\finpay_20260916_113427.sql
   ```
