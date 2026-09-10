package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gestion-ciclo-tres/internal/identity"
)

const userCols = `u.id, u.username, u.email, u.password, u.first_name, u.last_name,
		u.legajo, u.department, u.activo, u.created_at, u.updated_at,
		r.id, r.nombre_rol, r.nivel, r.descripcion, r.activo`

const userJoin = `FROM usuarios u LEFT JOIN roles r ON u.id_rol = r.id`

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
		SELECT `+userCols+`
		`+userJoin+`
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

func (r *SQLiteUserRepository) GetByID(ctx context.Context, id int64) (*identity.Usuario, error) {
	var u identity.Usuario
	var rolActivo int
	err := r.db.QueryRowContext(ctx, `
		SELECT `+userCols+`
		`+userJoin+`
		WHERE u.id = ?`, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&u.Legajo, &u.Department, &u.Activo, &u.CreatedAt, &u.UpdatedAt,
		&u.Rol.ID, &u.Rol.NombreRol, &u.Rol.Nivel, &u.Rol.Descripcion, &rolActivo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, identity.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("get by id %d: %w", id, err)
	}
	u.Rol.Activo = rolActivo != 0
	return &u, nil
}

func (r *SQLiteUserRepository) ListAll(ctx context.Context) ([]identity.Usuario, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+userCols+`
		`+userJoin+`
		ORDER BY u.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listar usuarios: %w", err)
	}
	defer rows.Close()

	var usuarios []identity.Usuario
	for rows.Next() {
		var u identity.Usuario
		var rolActivo int
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.Password, &u.FirstName, &u.LastName,
			&u.Legajo, &u.Department, &u.Activo, &u.CreatedAt, &u.UpdatedAt,
			&u.Rol.ID, &u.Rol.NombreRol, &u.Rol.Nivel, &u.Rol.Descripcion, &rolActivo,
		); err != nil {
			return nil, fmt.Errorf("scan usuario: %w", err)
		}
		u.Rol.Activo = rolActivo != 0
		usuarios = append(usuarios, u)
	}
	return usuarios, rows.Err()
}

func (r *SQLiteUserRepository) Update(ctx context.Context, u *identity.Usuario) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE usuarios
		SET first_name = ?, last_name = ?, email = ?, username = ?, legajo = ?,
		    department = ?, id_rol = ?, activo = ?, updated_at = datetime('now')
		WHERE id = ?`,
		u.FirstName, u.LastName, u.Email, u.Username, u.Legajo,
		u.Department, u.Rol.ID, u.Activo, u.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			if strings.Contains(err.Error(), "usuarios.username") {
				return identity.ErrUsernameExiste
			}
			if strings.Contains(err.Error(), "usuarios.email") {
				return identity.ErrEmailExiste
			}
		}
		return fmt.Errorf("actualizar usuario %d: %w", u.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected update usuario %d: %w", u.ID, err)
	}
	if n == 0 {
		return identity.ErrUsuarioNoEncontrado
	}
	return nil
}

func (r *SQLiteUserRepository) UpdatePassword(ctx context.Context, id int64, password string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE usuarios SET password = ?, updated_at = datetime('now') WHERE id = ?`,
		password, id)
	if err != nil {
		return fmt.Errorf("actualizar password usuario %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected update password %d: %w", id, err)
	}
	if n == 0 {
		return identity.ErrUsuarioNoEncontrado
	}
	return nil
}

func (r *SQLiteUserRepository) CountAll(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM usuarios`).Scan(&n); err != nil {
		return 0, fmt.Errorf("contar usuarios: %w", err)
	}
	return n, nil
}

func (r *SQLiteUserRepository) ExistsByUsernameOrEmailExcluding(ctx context.Context, username, email string, excludeID int64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM usuarios WHERE (username = ? OR email = ?) AND id != ? LIMIT 1`,
		username, email, excludeID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificar existencia de usuario excluyendo %d: %w", excludeID, err)
	}
	return true, nil
}
