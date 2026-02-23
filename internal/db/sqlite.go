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
	if err := os.MkdirAll(dir, 0o700); err != nil {
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

	// Migration v1 → v2: add agent_started_at, agent_spawned_status, reset_requested;
	// expand agent_status CHECK to include 'completed'
	if currentVersion < 2 {
		tx, txErr := d.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning v2 migration transaction: %w", txErr)
		}
		defer tx.Rollback()

		if _, txErr = tx.ExecContext(ctx, migrateV1toV2); txErr != nil {
			return fmt.Errorf("applying v2 migration: %w", txErr)
		}
		if _, txErr = tx.ExecContext(ctx,
			"INSERT OR REPLACE INTO schema_version (version) VALUES (2)"); txErr != nil {
			return fmt.Errorf("updating schema version to 2: %w", txErr)
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing v2 migration: %w", txErr)
		}
	}

	if currentVersion < 3 {
		tx, txErr := d.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning v3 migration transaction: %w", txErr)
		}
		defer tx.Rollback()

		if _, txErr = tx.ExecContext(ctx, migrateV2toV3); txErr != nil {
			return fmt.Errorf("applying v3 migration: %w", txErr)
		}
		if _, txErr = tx.ExecContext(ctx,
			"INSERT OR REPLACE INTO schema_version (version) VALUES (3)"); txErr != nil {
			return fmt.Errorf("updating schema version to 3: %w", txErr)
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing v3 migration: %w", txErr)
		}
	}

	return nil
}
