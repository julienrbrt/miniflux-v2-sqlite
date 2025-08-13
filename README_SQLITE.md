# Miniflux v2 - SQLite Edition

This is a modified version of Miniflux v2 that uses SQLite instead of PostgreSQL as the database backend.

## Changes Made

### Database Layer
- **Replaced PostgreSQL with SQLite**: Updated all database connections to use `github.com/glebarez/sqlite` instead of `github.com/lib/pq`
- **Updated SQL Syntax**: Converted PostgreSQL-specific SQL to SQLite-compatible syntax
- **Removed PostgreSQL-specific Features**: Eliminated features not supported by SQLite

### Key Migrations

#### Data Types
- `SERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`
- `BIGSERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`
- `BYTEA` → `BLOB`
- `TIMESTAMP WITH TIME ZONE` → `DATETIME`
- `INET` → `TEXT`
- `BOOLEAN` → `INTEGER` (0/1)
- PostgreSQL arrays → JSON strings

#### Functions and Features
- `now()` → `datetime('now')`
- `INTERVAL` → `datetime()` calculations
- Removed full-text search (`tsvector`, `to_tsvector`)
- Removed `HSTORE` extension
- `ON CONFLICT ... DO UPDATE` → `INSERT OR REPLACE`
- `RETURNING` clauses → separate queries

#### Indexes
- Removed GIN indexes
- Removed complex PostgreSQL-specific indexes
- Simplified to standard B-tree indexes

### Configuration Changes

#### Database Connection
SQLite uses file-based storage. The connection string should be a file path:
```
DATABASE_URL=./miniflux.db
```

#### Pragmas
The following SQLite pragmas are automatically set:
- `PRAGMA foreign_keys = ON` - Enable foreign key constraints
- `PRAGMA journal_mode = WAL` - Use Write-Ahead Logging for better concurrency
- `PRAGMA synchronous = NORMAL` - Balance performance and safety

### Limitations

#### Missing Features
1. **Full-Text Search**: PostgreSQL's advanced full-text search has been replaced with basic `LIKE` queries
2. **Complex Array Operations**: PostgreSQL arrays have been replaced with JSON strings
3. **Advanced Time Zone Support**: Simplified timezone handling
4. **Window Functions**: Older SQLite versions don't support window functions used in pagination

#### Performance Considerations
- SQLite is generally faster for small to medium datasets
- No network overhead (file-based)
- Limited concurrent write operations
- Single-writer, multiple-reader model

### Running the Application

1. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

2. **Build**:
   ```bash
   go build -o miniflux ./cmd/miniflux
   ```

3. **Initialize Database**:
   ```bash
   ./miniflux -migrate
   ```

4. **Create Admin User**:
   ```bash
   ./miniflux -create-admin
   ```

5. **Run**:
   ```bash
   ./miniflux
   ```

### Environment Variables

Key environment variables for SQLite version:

```bash
# Database
DATABASE_URL=./miniflux.db

# Server
LISTEN_ADDR=localhost:8080
BASE_URL=http://localhost:8080

# Optional: Backup location
BACKUP_DIR=./backups
```

### Backup and Restore

#### Backup
```bash
# Simple file copy (stop application first)
cp miniflux.db miniflux_backup_$(date +%Y%m%d_%H%M%S).db

# Or use SQLite backup command
sqlite3 miniflux.db ".backup miniflux_backup.db"
```

#### Restore
```bash
# Simple file copy
cp miniflux_backup.db miniflux.db

# Or use SQLite restore
sqlite3 miniflux.db ".restore miniflux_backup.db"
```

### Migration from PostgreSQL

To migrate from PostgreSQL to SQLite:

1. Export data from PostgreSQL
2. Convert schema and data format
3. Import into SQLite

**Note**: A direct migration tool is not provided. Manual data export/import may be required.

### Development

#### Testing
```bash
go test ./...
```

#### Database Schema
The SQLite schema is automatically created during migration. To view:
```bash
sqlite3 miniflux.db ".schema"
```

### Troubleshooting

#### Common Issues

1. **Database Locked**: Ensure only one instance is running
2. **Permission Errors**: Check file permissions on database file
3. **Performance Issues**: Consider using WAL mode (enabled by default)

#### Debug Mode
```bash
LOG_LEVEL=debug ./miniflux
```

### Contributing

When contributing to this SQLite version:

1. Test with SQLite instead of PostgreSQL
2. Avoid PostgreSQL-specific SQL syntax
3. Use parameterized queries with `?` placeholders
4. Handle boolean values as integers (0/1)

### License

Same as original Miniflux v2 - Apache 2.0 License
