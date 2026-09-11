package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Credenciales de los usuarios de ejemplo. Solo de desarrollo; F1 definirá el flujo real.
const (
	adminUser          = "admin"
	adminPassword      = "password"
	adminEmail         = "admin@empresa.com"
	produccionUser     = "produccion"
	produccionPassword = "produccion123"
	produccionEmail    = "produccion@empresa.com"
)

type seedUser struct {
	username   string
	email      string
	password   string
	firstName  string
	lastName   string
	legajo     string
	department string
	rolID      int64
}

var seedUsers = []seedUser{
	{username: adminUser, email: adminEmail, password: adminPassword,
		firstName: "Administrador", lastName: "Sistema", legajo: "A0001",
		department: "Administración", rolID: 1},
	{username: produccionUser, email: produccionEmail, password: produccionPassword,
		firstName: "Usuario", lastName: "Producción", legajo: "P0001",
		department: "Producción", rolID: 4},
}

// Seed inserta los datos de ejemplo si la base está vacía (idempotente).
// Los passwords se generan con bcrypt de Go; no hay hashes importados de PHP.
func Seed(ctx context.Context, db *sql.DB) error {
	roleCount, err := count(ctx, db, "roles")
	if err != nil {
		return fmt.Errorf("contar roles: %w", err)
	}
	if roleCount < 4 {
		return fmt.Errorf("seed: se esperaban 4 roles, hay %d (¿migración 0002_seed.sql aplicada?)", roleCount)
	}

	userCount, err := count(ctx, db, "usuarios")
	if err != nil {
		return fmt.Errorf("contar usuarios: %w", err)
	}
	if userCount > 0 {
		return nil
	}

	return seedAll(ctx, db)
}

func seedAll(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := insertUser(ctx, tx, seedUsers[0]); err != nil {
		return err
	}
	if _, err := insertUser(ctx, tx, seedUsers[1]); err != nil {
		return err
	}

	return tx.Commit()
}

func insertUser(ctx context.Context, tx *sql.Tx, u seedUser) (int64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("bcrypt %s: %w", u.username, err)
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO usuarios (username, email, password, first_name, last_name, legajo, department, id_rol)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.username, u.email, string(hash), u.firstName, u.lastName, u.legajo, u.department, u.rolID)
	if err != nil {
		return 0, fmt.Errorf("insertar usuario %s: %w", u.username, err)
	}
	return res.LastInsertId()
}

func count(ctx context.Context, db *sql.DB, table string) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
