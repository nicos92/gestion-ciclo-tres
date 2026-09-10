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

func TestUserGetByID_Exists(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.Username != "admin" {
		t.Errorf("username = %q, want admin", u.Username)
	}
	if u.Rol.Nivel != 4 {
		t.Errorf("nivel = %d, want 4", u.Rol.Nivel)
	}
}

func TestUserGetByID_NotFound(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	_, err := repo.GetByID(ctx, 99999)
	if err != identity.ErrUsuarioNoEncontrado {
		t.Fatalf("expected ErrUsuarioNoEncontrado, got %v", err)
	}
}

func TestListAll(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	list, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(list))
	}
	if list[0].Rol.NombreRol == "" {
		t.Error("expected rol to be populated")
	}
}

func TestUserUpdate_Success(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	u.FirstName = "Admin Updated"
	u.LastName = "Sistema Updated"
	u.Department = "Nuevo Depto"

	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if updated.FirstName != "Admin Updated" {
		t.Errorf("first_name = %q, want Admin Updated", updated.FirstName)
	}
	if updated.Department != "Nuevo Depto" {
		t.Errorf("department = %q, want Nuevo Depto", updated.Department)
	}
}

func TestUserUpdate_NotFound(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u := &identity.Usuario{
		ID: 99999, Username: "noexiste", Email: "no@test.com",
		FirstName: "No", LastName: "Existe", Legajo: "N0001",
		Department: "Test", Rol: identity.Rol{ID: 1}, Activo: true,
	}
	err := repo.Update(ctx, u)
	if err != identity.ErrUsuarioNoEncontrado {
		t.Fatalf("expected ErrUsuarioNoEncontrado, got %v", err)
	}
}

func TestUpdate_Duplicado(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	u, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	u.Username = "produccion"
	err = repo.Update(ctx, u)
	if err != identity.ErrUsernameExiste {
		t.Fatalf("expected ErrUsernameExiste, got %v", err)
	}
}

func TestUpdatePassword_Success(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	err := repo.UpdatePassword(ctx, 1, "nuevohash123")
	if err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	u, err := repo.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.Password != "nuevohash123" {
		t.Errorf("password = %q, want nuevohash123", u.Password)
	}
}

func TestUpdatePassword_NotFound(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	err := repo.UpdatePassword(ctx, 99999, "hash")
	if err != identity.ErrUsuarioNoEncontrado {
		t.Fatalf("expected ErrUsuarioNoEncontrado, got %v", err)
	}
}

func TestCountAll(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	count, err := repo.CountAll(ctx)
	if err != nil {
		t.Fatalf("CountAll: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2, got %d", count)
	}
}

func TestExistsByUsernameOrEmailExcluding(t *testing.T) {
	ctx, tdb := newTestDB(t)
	repo := NewUserRepository(tdb.db)

	exists, err := repo.ExistsByUsernameOrEmailExcluding(ctx, "admin", "admin@empresa.com", 1)
	if err != nil {
		t.Fatalf("ExistsByUsernameOrEmailExcluding: %v", err)
	}
	if exists {
		t.Error("expected false when excluding own ID")
	}

	exists, err = repo.ExistsByUsernameOrEmailExcluding(ctx, "admin", "admin@empresa.com", 2)
	if err != nil {
		t.Fatalf("ExistsByUsernameOrEmailExcluding: %v", err)
	}
	if !exists {
		t.Error("expected true when excluding different ID")
	}
}
