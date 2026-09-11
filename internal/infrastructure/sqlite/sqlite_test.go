package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// openTestDB abre una DB en t.TempDir() y la cierra al finalizar el test.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// realMigrationsFS lee las migraciones reales de migrations/ (sin duplicarlas).
func realMigrationsFS(t *testing.T) fs.FS {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller falló")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return os.DirFS(filepath.Join(root, "migrations"))
}

func TestOpenCreatesDirAndPragma(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "db.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !strings.Contains(DSN(path), "journal_mode(WAL)") ||
		!strings.Contains(DSN(path), "foreign_keys(1)") ||
		!strings.Contains(DSN(path), "_txlock=immediate") {
		t.Errorf("DSN %q sin las pragmas esperadas", DSN(path))
	}
	if strings.Contains(DSN(path), `\`) {
		t.Errorf("DSN con backslashes (Windows): %q", DSN(path))
	}
}

func TestMigrateAppliesMigrations(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := Migrate(ctx, db, realMigrationsFS(t)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var versionCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&versionCount); err != nil {
		t.Fatalf("contar versiones: %v", err)
	}
	if versionCount != 4 {
		t.Errorf("versiones aplicadas = %d, se esperaban 4", versionCount)
	}

	for _, table := range []string{"roles", "usuarios", "tarimas", "historial_tarimas"} {
		var n int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil {
			t.Fatalf("buscar tabla %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("no existe la tabla %s", table)
		}
	}

	var viewCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name='vista_tarimas_con_legajo'`).Scan(&viewCount); err != nil {
		t.Fatalf("buscar vista: %v", err)
	}
	if viewCount != 1 {
		t.Errorf("no existe la vista vista_tarimas_con_legajo")
	}

	var triggerCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='trg_historial_tarimas'`).Scan(&triggerCount); err != nil {
		t.Fatalf("buscar trigger: %v", err)
	}
	if triggerCount != 1 {
		t.Errorf("no existe el trigger trg_historial_tarimas")
	}

	var roles int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM roles`).Scan(&roles); err != nil {
		t.Fatalf("contar roles: %v", err)
	}
	if roles != 4 {
		t.Errorf("roles = %d, se esperaban 4", roles)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	tfs := realMigrationsFS(t)

	if err := Migrate(ctx, db, tfs); err != nil {
		t.Fatalf("Migrate 1ra vez: %v", err)
	}
	if err := Migrate(ctx, db, tfs); err != nil {
		t.Fatalf("Migrate 2da vez: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("contar versiones: %v", err)
	}
	if count != 4 {
		t.Errorf("versiones = %d tras re-ejecutar, se esperaban 4", count)
	}
}
