package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) a SQLite database using the configured URL.
// Supported formats:
//   - sqlite3:./data.db
//   - sqlite:./data.db
//   - file:./data.db
func Open(databaseURL string) (*sql.DB, error) {
	dsn := normalizeDSN(databaseURL)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// SQLite works best with a single writer connection for WAL
	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetMaxIdleConns(1)

	if err := configurePragmas(db); err != nil {
		return nil, err
	}

	if err := ensureSchema(db); err != nil {
		return nil, err
	}

	return db, nil
}

func normalizeDSN(databaseURL string) string {
	dsn := strings.TrimSpace(databaseURL)
	if dsn == "" {
		dsn = "./data.db"
	}

	if idx := strings.Index(dsn, ":"); idx != -1 {
		prefix := dsn[:idx]
		if prefix == "sqlite3" || prefix == "sqlite" {
			dsn = dsn[idx+1:]
		}
	}

	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		dsn = "./data.db"
	}

	if !strings.HasPrefix(dsn, "file:") {
		if !strings.Contains(dsn, ":/") && !strings.HasPrefix(dsn, "./") && !strings.HasPrefix(dsn, "/") {
			dsn = "./" + dsn
		}
		dsn = "file:" + filepath.Clean(dsn)
	}

	if !strings.Contains(dsn, "?") {
		dsn += "?_pragma=busy_timeout(5000)"
	}

	return dsn
}

func configurePragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("configure sqlite pragma (%s): %w", pragma, err)
		}
	}
	return nil
}

func ensureSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			youtube_channel_id TEXT NOT NULL UNIQUE,
			tiktok_account_id TEXT NOT NULL UNIQUE,
			tiktok_access_token TEXT NOT NULL,
			tiktok_refresh_token TEXT,
			tiktok_token_expires_at TIMESTAMP NULL,
			last_checked_at TIMESTAMP NULL,
			last_video_id TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY,
			youtube_video_id TEXT NOT NULL UNIQUE,
			account_id TEXT NOT NULL,
			title TEXT,
			description TEXT,
			thumbnail_url TEXT,
			video_url TEXT,
			local_file_path TEXT,
			status TEXT NOT NULL,
			error_message TEXT,
			tiktok_video_id TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			published_at TIMESTAMP,
			FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_videos_status_created ON videos(status, created_at);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}

	// Add new columns if they don't exist (for existing databases)
	// SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we need to check first
	migrationStatements := []struct {
		checkQuery string
		addQuery   string
	}{
		{
			checkQuery: `SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name='tiktok_refresh_token'`,
			addQuery:   `ALTER TABLE accounts ADD COLUMN tiktok_refresh_token TEXT`,
		},
		{
			checkQuery: `SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name='tiktok_token_expires_at'`,
			addQuery:   `ALTER TABLE accounts ADD COLUMN tiktok_token_expires_at TIMESTAMP NULL`,
		},
	}

	for _, migration := range migrationStatements {
		var count int
		err := db.QueryRow(migration.checkQuery).Scan(&count)
		if err != nil {
			// If query fails, try to add column anyway (table might exist but pragma query failed)
			_, _ = db.Exec(migration.addQuery)
			continue
		}
		if count == 0 {
			// Column doesn't exist, add it
			_, err = db.Exec(migration.addQuery)
			// Ignore error - column might already exist due to race condition
			// or SQLite might return error if column exists, which is fine
			_ = err
		}
	}

	return nil
}
