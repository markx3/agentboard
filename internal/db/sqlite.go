package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Critical: single connection to prevent SQLITE_BUSY
	conn.SetMaxOpenConns(1)

	if err := applyPragmas(conn); err != nil {
		conn.Close()
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(context.Background()); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

func applyPragmas(conn *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -8000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			return fmt.Errorf("applying pragma %q: %w", p, err)
		}
	}
	return nil
}

func (d *DB) migrate(ctx context.Context) error {
	var currentVersion int
	err := d.conn.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		// Table doesn't exist yet, run full schema
		if _, err := d.conn.ExecContext(ctx, schemaSQL); err != nil {
			return fmt.Errorf("creating schema: %w", err)
		}
		_, err = d.conn.ExecContext(ctx,
			"INSERT INTO schema_version (version) VALUES (?)", schemaVersion)
		return err
	}

	if currentVersion >= schemaVersion {
		return nil
	}

	// Future migrations would go here
	return nil
}
