package identity

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type mockRepo struct {
	users    map[string]*Usuario
	userByID map[int64]*Usuario
	exists   bool
	createID int64
	nextID   int64
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generar hash: %v", err)
	}
	return string(hash)
}

func newMockRepo(t *testing.T) *mockRepo {
	t.Helper()
	admin := &Usuario{
		ID: 1, Username: "admin", Email: "admin@empresa.com",
		Password:  hashPassword(t, "secret"),
		FirstName: "Admin", LastName: "Sistema", Legajo: "A0001",
		Department: "Admin", Activo: true,
		Rol: Rol{ID: 1, NombreRol: "administrador", Nivel: 4, Activo: true},
	}
	inactivo := &Usuario{
		ID: 2, Username: "inactivo", Email: "inactivo@test.com",
		Password:  hashPassword(t, "secret"),
		FirstName: "In", LastName: "Activo", Legajo: "I0001",
		Department: "Test", Activo: false,
		Rol: Rol{ID: 4, NombreRol: "produccion", Nivel: 1, Activo: true},
	}
	return &mockRepo{
		users:    map[string]*Usuario{"admin": admin, "inactivo": inactivo},
		userByID: map[int64]*Usuario{1: admin, 2: inactivo},
		nextID:   3,
	}
}

func (m *mockRepo) GetByUsername(_ context.Context, username string) (*Usuario, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, ErrUsuarioNoEncontrado
	}
	return u, nil
}

func (m *mockRepo) Create(_ context.Context, u *Usuario) (int64, error) {
	u.ID = m.nextID
	m.nextID++
	m.users[u.Username] = u
	m.userByID[u.ID] = u
	return u.ID, nil
}

func (m *mockRepo) ExistsByUsernameOrEmail(_ context.Context, username, email string) (bool, error) {
	if m.exists {
		return true, nil
	}
	for _, u := range m.users {
		if u.Username == username || u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepo) GetByID(_ context.Context, id int64) (*Usuario, error) {
	u, ok := m.userByID[id]
	if !ok {
		return nil, ErrUsuarioNoEncontrado
	}
	return u, nil
}

func (m *mockRepo) ListAll(_ context.Context) ([]Usuario, error) {
	var list []Usuario
	for _, u := range m.users {
		list = append(list, *u)
	}
	return list, nil
}

func (m *mockRepo) Update(_ context.Context, u *Usuario) error {
	existing, ok := m.userByID[u.ID]
	if !ok {
		return ErrUsuarioNoEncontrado
	}
	existing.FirstName = u.FirstName
	existing.LastName = u.LastName
	existing.Email = u.Email
	existing.Username = u.Username
	existing.Legajo = u.Legajo
	existing.Department = u.Department
	existing.Rol = u.Rol
	existing.Activo = u.Activo
	return nil
}

func (m *mockRepo) UpdatePassword(_ context.Context, id int64, password string) error {
	u, ok := m.userByID[id]
	if !ok {
		return ErrUsuarioNoEncontrado
	}
	u.Password = password
	return nil
}

func (m *mockRepo) CountAll(_ context.Context) (int, error) {
	return len(m.users), nil
}

func (m *mockRepo) ExistsByUsernameOrEmailExcluding(_ context.Context, username, email string, excludeID int64) (bool, error) {
	for _, u := range m.users {
		if u.ID == excludeID {
			continue
		}
		if u.Username == username || u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func TestLogin_OK(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)

	u, err := svc.Login(context.Background(), "admin", "secret")
	if err != nil {
		t.Fatalf("Login OK: %v", err)
	}
	if u.Username != "admin" {
		t.Errorf("username = %q, want admin", u.Username)
	}
}

func TestLogin_PasswordIncorrecta(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)

	_, err := svc.Login(context.Background(), "admin", "wrong")
	if err != ErrCredencialesInvalidas {
		t.Fatalf("expected ErrCredencialesInvalidas, got %v", err)
	}
}

func TestLogin_UsuarioNoExiste(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)

	_, err := svc.Login(context.Background(), "noexiste", "pass")
	if err != ErrUsuarioNoEncontrado {
		t.Fatalf("expected ErrUsuarioNoEncontrado, got %v", err)
	}
}

func TestLogin_UsuarioInactivo(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)

	_, err := svc.Login(context.Background(), "inactivo", "secret")
	if err != ErrUsuarioInactivo {
		t.Fatalf("expected ErrUsuarioInactivo, got %v", err)
	}
}

func TestLogin_CamposVacios(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))

	_, err := svc.Login(context.Background(), "", "")
	if err != ErrCamposRequeridos {
		t.Fatalf("expected ErrCamposRequeridos, got %v", err)
	}
}

func TestRegister_OK(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)

	u := &Usuario{
		Username:   "nuevo",
		Email:      "nuevo@test.com",
		FirstName:  "Nuevo",
		LastName:   "User",
		Legajo:     "N0001",
		Department: "Test",
		Rol:        Rol{ID: 4},
		Activo:     true,
	}
	id, err := svc.Register(context.Background(), u, "password123")
	if err != nil {
		t.Fatalf("Register OK: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestRegister_CamposVacios(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))

	u := &Usuario{Username: ""}
	_, err := svc.Register(context.Background(), u, "pass")
	if err != ErrCamposRequeridos {
		t.Fatalf("expected ErrCamposRequeridos, got %v", err)
	}
}

func TestRegister_EmailInvalido(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))

	u := &Usuario{
		Username: "test", Email: "noesemail",
		FirstName: "Test", LastName: "User",
		Legajo: "T0001", Department: "Test",
	}
	_, err := svc.Register(context.Background(), u, "pass")
	if err != ErrEmailInvalido {
		t.Fatalf("expected ErrEmailInvalido, got %v", err)
	}
}

func TestRegister_Duplicado(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)

	u := &Usuario{
		Username: "admin", Email: "admin@empresa.com",
		FirstName: "Dup", LastName: "User",
		Legajo: "D0001", Department: "Test",
	}
	_, err := svc.Register(context.Background(), u, "pass")
	if err != ErrUsernameExiste {
		t.Fatalf("expected ErrUsernameExiste, got %v", err)
	}
}

func TestGetByID_OK(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	u, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID OK: %v", err)
	}
	if u.Username != "admin" {
		t.Errorf("username = %q, want admin", u.Username)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	_, err := svc.GetByID(context.Background(), 999)
	if err != ErrUsuarioNoEncontrado {
		t.Fatalf("expected ErrUsuarioNoEncontrado, got %v", err)
	}
}

func TestListAll(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	list, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(list))
	}
}

func TestUpdate_OK(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	u := &Usuario{
		ID:         1,
		Username:   "admin",
		Email:      "admin@empresa.com",
		FirstName:  "Admin Updated",
		LastName:   "Sistema Updated",
		Legajo:     "A0001",
		Department: "Admin",
		Rol:        Rol{ID: 1},
		Activo:     true,
	}
	if err := svc.Update(context.Background(), u, ""); err != nil {
		t.Fatalf("Update OK: %v", err)
	}
	updated, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if updated.FirstName != "Admin Updated" {
		t.Errorf("first_name = %q, want Admin Updated", updated.FirstName)
	}
}

func TestUpdate_CamposVacios(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	u := &Usuario{ID: 1, Username: "", Email: "test@test.com",
		FirstName: "Test", LastName: "User", Legajo: "T0001", Department: "Test"}
	err := svc.Update(context.Background(), u, "")
	if err != ErrCamposRequeridos {
		t.Fatalf("expected ErrCamposRequeridos, got %v", err)
	}
}

func TestUpdate_EmailInvalido(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	u := &Usuario{ID: 1, Username: "admin", Email: "noesemail",
		FirstName: "Admin", LastName: "Sistema", Legajo: "A0001", Department: "Admin"}
	err := svc.Update(context.Background(), u, "")
	if err != ErrEmailInvalido {
		t.Fatalf("expected ErrEmailInvalido, got %v", err)
	}
}

func TestUpdate_Duplicado(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	u := &Usuario{ID: 1, Username: "inactivo", Email: "admin@empresa.com",
		FirstName: "Admin", LastName: "Sistema", Legajo: "A0001", Department: "Admin"}
	err := svc.Update(context.Background(), u, "")
	if err != ErrUsernameExiste {
		t.Fatalf("expected ErrUsernameExiste, got %v", err)
	}
}

func TestUpdate_ConPassword(t *testing.T) {
	repo := newMockRepo(t)
	svc := NewAuthService(repo)
	u := &Usuario{
		ID:         1,
		Username:   "admin",
		Email:      "admin@empresa.com",
		FirstName:  "Admin",
		LastName:   "Sistema",
		Legajo:     "A0001",
		Department: "Admin",
		Rol:        Rol{ID: 1},
		Activo:     true,
	}
	if err := svc.Update(context.Background(), u, "newpassword123"); err != nil {
		t.Fatalf("Update con password: %v", err)
	}
	updated, _ := svc.GetByID(context.Background(), 1)
	if updated.Password == "" {
		t.Error("password should be updated")
	}
}

func TestUpdate_SinPassword(t *testing.T) {
	repo := newMockRepo(t)
	original := hashPassword(t, "secret")
	repo.users["admin"].Password = original
	svc := NewAuthService(repo)
	u := &Usuario{
		ID:         1,
		Username:   "admin",
		Email:      "admin@empresa.com",
		FirstName:  "Admin New",
		LastName:   "Sistema",
		Legajo:     "A0001",
		Department: "Admin",
		Rol:        Rol{ID: 1},
		Activo:     true,
	}
	if err := svc.Update(context.Background(), u, ""); err != nil {
		t.Fatalf("Update sin password: %v", err)
	}
	updated, _ := svc.GetByID(context.Background(), 1)
	if updated.Password != original {
		t.Error("password should not change when newPassword is empty")
	}
}

func TestCountAll(t *testing.T) {
	svc := NewAuthService(newMockRepo(t))
	count, err := svc.CountAll(context.Background())
	if err != nil {
		t.Fatalf("CountAll: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2, got %d", count)
	}
}
