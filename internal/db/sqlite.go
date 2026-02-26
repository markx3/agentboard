package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
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

	// Use URI format with _txlock=immediate to acquire RESERVED lock at BEGIN
	// instead of waiting until the first write, preventing SQLITE_BUSY under
	// concurrent access.
	dsn := fmt.Sprintf("file:%s?_txlock=immediate", url.PathEscape(dbPath))
	conn, err := sql.Open("sqlite", dsn)
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

// hasColumn checks whether the given table has a column with the given name.
func hasColumn(ctx context.Context, conn *sql.DB, table, column string) (bool, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("checking column %s.%s: %w", table, column, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// tableExists checks whether the given table exists in the database.
func tableExists(ctx context.Context, conn *sql.DB, table string) (bool, error) {
	var count int
	err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking table %s: %w", table, err)
	}
	return count > 0, nil
}

// applyMigration executes migrationSQL inside tx and updates schema_version to version.
func applyMigration(ctx context.Context, tx *sql.Tx, version int, migrationSQL string) error {
	if _, err := tx.ExecContext(ctx, migrationSQL); err != nil {
		return fmt.Errorf("applying v%d migration: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT OR REPLACE INTO schema_version (version) VALUES (?)", version); err != nil {
		return fmt.Errorf("updating schema version to %d: %w", version, err)
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

	// Migration v1 -> v2: add agent_started_at, agent_spawned_status, reset_requested;
	// expand agent_status CHECK to include 'completed'
	if currentVersion < 2 {
		tx, txErr := d.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning v2 migration transaction: %w", txErr)
		}
		defer tx.Rollback()
		if txErr = applyMigration(ctx, tx, 2, migrateV1toV2); txErr != nil {
			return txErr
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
		if txErr = applyMigration(ctx, tx, 3, migrateV2toV3); txErr != nil {
			return txErr
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing v3 migration: %w", txErr)
		}
	}

	if currentVersion < 4 {
		tx, txErr := d.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning v4 migration transaction: %w", txErr)
		}
		defer tx.Rollback()
		if txErr = applyMigration(ctx, tx, 4, migrateV3toV4); txErr != nil {
			return txErr
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing v4 migration: %w", txErr)
		}
	}

	if currentVersion < 5 {
		if err := d.migrateV4toV5(ctx); err != nil {
			return err
		}
	}

	if currentVersion < 6 {
		tx, txErr := d.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning v6 migration transaction: %w", txErr)
		}
		defer tx.Rollback()
		if txErr = applyMigration(ctx, tx, 6, migrateV5toV6SQL); txErr != nil {
			return txErr
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing v6 migration: %w", txErr)
		}
	}

	if currentVersion < 7 {
		if err := d.migrateV6toV7(ctx); err != nil {
			return err
		}
	}

	return nil
}

// migrateV4toV5 handles the v5 migration with PRAGMA foreign_keys=OFF
// executed OUTSIDE the transaction (SQLite silently ignores it inside).
func (d *DB) migrateV4toV5(ctx context.Context) error {
	// Disable foreign keys outside the transaction
	if _, err := d.conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return fmt.Errorf("disabling foreign keys for v5 migration: %w", err)
	}

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		// Re-enable foreign keys before returning
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("beginning v5 migration transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, migrateV4toV5SQL); err != nil {
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("applying v5 migration: %w", err)
	}
	if _, err = tx.ExecContext(ctx,
		"INSERT OR REPLACE INTO schema_version (version) VALUES (5)"); err != nil {
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("updating schema version to 5: %w", err)
	}
	if err = tx.Commit(); err != nil {
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("committing v5 migration: %w", err)
	}

	// Re-enable foreign keys after migration
	if _, err := d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("re-enabling foreign keys after v5 migration: %w", err)
	}

	return nil
}

// migrateV6toV7 handles databases coming from either migration path:
// - HEAD path (v5): already has enrichment, suggestions, depends_on -- just needs agent_activity (added in v6)
// - Main path (v6): has agent_activity + blocks_id deps -- needs enrichment cols, suggestions table, deps conversion
// Each step checks for existence before acting, making it idempotent.
//
// IMPORTANT: All schema inspection (hasColumn, tableExists) is done BEFORE
// starting the transaction because SetMaxOpenConns(1) means the tx holds the
// only connection and any d.conn query would deadlock.
func (d *DB) migrateV6toV7(ctx context.Context) error {
	// Inspect schema BEFORE starting the transaction to avoid deadlock
	// with the single-connection pool.
	hasEnrichment, err := hasColumn(ctx, d.conn, "tasks", "enrichment_status")
	if err != nil {
		return fmt.Errorf("checking enrichment_status column: %w", err)
	}

	hasSuggestions, err := tableExists(ctx, d.conn, "suggestions")
	if err != nil {
		return fmt.Errorf("checking suggestions table: %w", err)
	}

	hasDepsTable, err := tableExists(ctx, d.conn, "task_dependencies")
	if err != nil {
		return fmt.Errorf("checking task_dependencies table: %w", err)
	}

	var hasBlocksID bool
	if hasDepsTable {
		hasBlocksID, err = hasColumn(ctx, d.conn, "task_dependencies", "blocks_id")
		if err != nil {
			return fmt.Errorf("checking blocks_id column: %w", err)
		}
	}

	// Disable foreign keys for potential table rebuild (deps conversion)
	if _, err := d.conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return fmt.Errorf("disabling foreign keys for v7 migration: %w", err)
	}

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("beginning v7 migration transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Add enrichment columns if missing (main-path DBs)
	if !hasEnrichment {
		if _, err = tx.ExecContext(ctx, migrateV6toV7SQL_addEnrichmentCols); err != nil {
			d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
			return fmt.Errorf("adding enrichment columns in v7: %w", err)
		}
	}

	// Step 2: Create suggestions table if missing (main-path DBs)
	if !hasSuggestions {
		if _, err = tx.ExecContext(ctx, migrateV6toV7SQL_createSuggestions); err != nil {
			d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
			return fmt.Errorf("creating suggestions table in v7: %w", err)
		}
	}

	// Step 3: Convert task_dependencies from blocks_id to depends_on if needed (main-path DBs)
	if hasDepsTable {
		if hasBlocksID {
			if _, err = tx.ExecContext(ctx, migrateV6toV7SQL_convertDeps); err != nil {
				d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
				return fmt.Errorf("converting dependencies in v7: %w", err)
			}
		}
	} else {
		// No deps table at all -- create it with depends_on naming
		if _, err = tx.ExecContext(ctx, `
			CREATE TABLE task_dependencies (
			    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			    depends_on TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			    created_at TEXT NOT NULL DEFAULT (datetime('now')),
			    PRIMARY KEY (task_id, depends_on),
			    CHECK(task_id != depends_on)
			);
			CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on);
		`); err != nil {
			d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
			return fmt.Errorf("creating task_dependencies in v7: %w", err)
		}
	}

	if _, err = tx.ExecContext(ctx,
		"INSERT OR REPLACE INTO schema_version (version) VALUES (7)"); err != nil {
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("updating schema version to 7: %w", err)
	}
	if err = tx.Commit(); err != nil {
		d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")
		return fmt.Errorf("committing v7 migration: %w", err)
	}

	// Re-enable foreign keys
	if _, err := d.conn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("re-enabling foreign keys after v7 migration: %w", err)
	}

	return nil
}
