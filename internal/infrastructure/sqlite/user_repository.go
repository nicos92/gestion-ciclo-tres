package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gestion-ciclo-tres/internal/identity"
)

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) GetByUsername(ctx context.Context, username string) (*identity.Usuario, error) {
	var u identity.Usuario
	var rolActivo int
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.email, u.password, u.first_name, u.last_name,
		       u.legajo, u.department, u.activo, u.created_at, u.updated_at,
		       r.id, r.nombre_rol, r.nivel, r.descripcion, r.activo
		FROM usuarios u
		LEFT JOIN roles r ON u.id_rol = r.id
		WHERE u.username = ?`, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.Legajo, &u.Department, &u.Activo, &u.CreatedAt, &u.UpdatedAt,
		&u.Rol.ID, &u.Rol.NombreRol, &u.Rol.Nivel, &u.Rol.Descripcion, &rolActivo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, identity.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("get by username %q: %w", username, err)
	}
	u.Rol.Activo = rolActivo != 0
	return &u, nil
}

func (r *SQLiteUserRepository) Create(ctx context.Context, u *identity.Usuario) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO usuarios (username, email, password, first_name, last_name, legajo, department, id_rol, activo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.Email, u.Password, u.FirstName, u.LastName,
		u.Legajo, u.Department, u.Rol.ID, u.Activo,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			if strings.Contains(err.Error(), "usuarios.username") {
				return 0, identity.ErrUsernameExiste
			}
			if strings.Contains(err.Error(), "usuarios.email") {
				return 0, identity.ErrEmailExiste
			}
		}
		return 0, fmt.Errorf("insertar usuario %q: %w", u.Username, err)
	}
	return res.LastInsertId()
}

func (r *SQLiteUserRepository) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM usuarios WHERE username = ? OR email = ? LIMIT 1`,
		username, email).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificar existencia de usuario: %w", err)
	}
	return true, nil
}
