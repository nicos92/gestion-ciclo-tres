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

type seedTarima struct {
	numeroProducto string
	numeroTarima   string
	numeroUsuario  string
	conservacion   string
	cajas          int
	peso           float64
	numeroVenta    string
	descripcion    string
}

var seedTarimas = []seedTarima{
	{numeroProducto: "880197", numeroTarima: "000999", numeroUsuario: "010",
		conservacion: "1", cajas: 45, peso: 450.00, numeroVenta: "25-123456",
		descripcion: "Tarima de ejemplo 1"},
	{numeroProducto: "880197", numeroTarima: "001000", numeroUsuario: "011",
		conservacion: "2", cajas: 30, peso: 300.00, numeroVenta: "25-654321",
		descripcion: "Tarima de ejemplo 2"},
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

	adminID, err := insertUser(ctx, tx, seedUsers[0])
	if err != nil {
		return err
	}
	if _, err := insertUser(ctx, tx, seedUsers[1]); err != nil {
		return err
	}
	for _, t := range seedTarimas {
		if err := insertTarima(ctx, tx, t, adminID); err != nil {
			return err
		}
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

func insertTarima(ctx context.Context, tx *sql.Tx, t seedTarima, idUsuario int64) error {
	codigoBarras, err := buildBarcode(t.numeroProducto, t.numeroTarima, t.numeroUsuario, t.conservacion, t.cajas, t.peso)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tarimas (codigo_barras, numero_producto, numero_tarima, numero_usuario,
		                     conservacion, cantidad_cajas, peso, numero_venta, descripcion, id_usuario)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		codigoBarras, t.numeroProducto, t.numeroTarima, t.numeroUsuario,
		t.conservacion, t.cajas, t.peso, t.numeroVenta, t.descripcion, idUsuario)
	if err != nil {
		return fmt.Errorf("insertar tarima %s: %w", t.numeroTarima, err)
	}
	return nil
}

// buildBarcode arma un código de barras de 30 dígitos (formato de la app):
// 1 dígito "0" + 6 de producto + 6 de tarima + "9998" + 1 de conservación + 3 de usuario + 3 de cajas + 6 de peso.
func buildBarcode(numeroProducto, numeroTarima, numeroUsuario, conservacion string, cajas int, peso float64) (string, error) {
	pesoCentavos := int(peso*100 + 0.5)
	barcode := "0" +
		numeroProducto + numeroTarima +
		"9998" + conservacion + numeroUsuario +
		fmt.Sprintf("%03d", cajas) +
		fmt.Sprintf("%06d", pesoCentavos)
	if len(barcode) != 30 {
		return "", fmt.Errorf("barcode de %d dígitos, se esperaban 30", len(barcode))
	}
	return barcode, nil
}

func count(ctx context.Context, db *sql.DB, table string) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
