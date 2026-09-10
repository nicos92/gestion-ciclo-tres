// Package sqlite gestiona la conexión, migración y seed de la base de datos SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	driverName   = "sqlite"
	maxOpenConns = 4
	maxIdleConns = 4
)

// dsnQuery son pragmas aplicados por el driver modernc en cada conexión.
// WAL + busy_timeout + _txlock=immediate permiten un pool >1 sin SQLITE_BUSY.
// Si en fases posteriores aparecen escrituras bloqueadas, el fallback
// documentado es db.SetMaxOpenConns(1).
const dsnQuery = "?_pragma=foreign_keys(1)" +
	"&_pragma=busy_timeout(5000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_txlock=immediate"

// Open abre (o crea si no existe) la base SQLite en path, aplicando el DSN
// con WAL, foreign keys y timeout de escritura. Crea el directorio padre si falta.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("crear directorio %q: %w", dir, err)
		}
	}

	db, err := sql.Open(driverName, DSN(path))
	if err != nil {
		return nil, fmt.Errorf("abrir sqlite %q: %w", path, err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	return db, nil
}

// DSN devuelve el DSN completo para path, normalizando las barras invertidas
// de Windows a forward slashes para que funcione igual en Linux/contenedor.
func DSN(path string) string {
	return "file:" + filepath.ToSlash(path) + dsnQuery
}
