package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    TEXT PRIMARY KEY,
	applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

// Migrate aplica los archivos *.sql de fsys (orden lexicográfico) que aún no
// estén registrados en schema_migrations. Cada migración corre en su propia
// transacción y se registra su versión al completarse. Es idempotente.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if _, err := db.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("crear schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("leer directorio de migraciones: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if applied[name] {
			continue
		}
		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("leer %s: %w", name, err)
		}
		if err := applyMigration(ctx, db, name, string(content)); err != nil {
			return fmt.Errorf("aplicar %s: %w", name, err)
		}
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("listar schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("leer version: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar schema_migrations: %w", err)
	}
	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, version, content string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, content); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
		return err
	}
	return tx.Commit()
}
