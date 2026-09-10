package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"gestion-ciclo-tres/internal/identity"
)

func newTestDB(t *testing.T) (context.Context, *testDBConn) {
	t.Helper()
	db := openTestDB(t)
	ctx := context.Background()
	if err := Migrate(ctx, db, realMigrationsFS(t)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return ctx, &testDBConn{db: db}
}

type testDBConn struct {
	db *sql.DB
}

func TestGetByUsername_Admin(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByUsername admin: %v", err)
	}
	if u.Username != "admin" {
		t.Errorf("username = %q, want admin", u.Username)
	}
	if u.Rol.NombreRol != "administrador" {
		t.Errorf("rol = %q, want administrador", u.Rol.NombreRol)
	}
	if u.Rol.Nivel != 4 {
		t.Errorf("nivel = %d, want 4", u.Rol.Nivel)
	}
	if !u.Activo {
		t.Error("admin should be active")
	}
}

func TestGetByUsername_Produccion(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u, err := repo.GetByUsername(ctx, "produccion")
	if err != nil {
		t.Fatalf("GetByUsername produccion: %v", err)
	}
	if u.Rol.NombreRol != "produccion" {
		t.Errorf("rol = %q, want produccion", u.Rol.NombreRol)
	}
	if u.Rol.Nivel != 1 {
		t.Errorf("nivel = %d, want 1", u.Rol.Nivel)
	}
}

func TestGetByUsername_NoExiste(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	_, err := repo.GetByUsername(ctx, "noexiste")
	if err != identity.ErrUsuarioNoEncontrado {
		t.Fatalf("expected ErrUsuarioNoEncontrado, got %v", err)
	}
}

func TestCreate(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u := &identity.Usuario{
		Username:   "nuevo",
		Email:      "nuevo@test.com",
		Password:   "hashfalso",
		FirstName:  "Nuevo",
		LastName:   "Usuario",
		Legajo:     "N0001",
		Department: "Testing",
		Rol:        identity.Rol{ID: 4},
		Activo:     true,
	}
	id, err := repo.Create(ctx, u)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID")
	}

	fetched, err := repo.GetByUsername(ctx, "nuevo")
	if err != nil {
		t.Fatalf("GetByUsername after create: %v", err)
	}
	if fetched.Email != "nuevo@test.com" {
		t.Errorf("email = %q, want nuevo@test.com", fetched.Email)
	}
}

func TestCreate_Duplicado(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u := &identity.Usuario{
		Username:   "admin",
		Email:      "otro@test.com",
		Password:   "hash",
		FirstName:  "Dup",
		LastName:   "User",
		Legajo:     "D0001",
		Department: "Test",
		Rol:        identity.Rol{ID: 4},
		Activo:     true,
	}
	_, err := repo.Create(ctx, u)
	if err != identity.ErrUsernameExiste {
		t.Fatalf("expected ErrUsernameExiste, got %v", err)
	}
}

func TestExistsByUsernameOrEmail(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	exists, err := repo.ExistsByUsernameOrEmail(ctx, "admin", "admin@empresa.com")
	if err != nil {
		t.Fatalf("ExistsByUsernameOrEmail: %v", err)
	}
	if !exists {
		t.Error("expected true for existing user")
	}

	exists, err = repo.ExistsByUsernameOrEmail(ctx, "noexiste", "noexiste@test.com")
	if err != nil {
		t.Fatalf("ExistsByUsernameOrEmail: %v", err)
	}
	if exists {
		t.Error("expected false for non-existing user")
	}
}
