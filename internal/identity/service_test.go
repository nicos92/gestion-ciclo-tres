package identity

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type mockRepo struct {
	users    map[string]*Usuario
	exists   bool
	createID int64
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
	return &mockRepo{
		users: map[string]*Usuario{
			"admin": {
				ID: 1, Username: "admin", Email: "admin@empresa.com",
				Password:  hashPassword(t, "secret"),
				FirstName: "Admin", LastName: "Sistema", Legajo: "A0001",
				Department: "Admin", Activo: true,
				Rol: Rol{ID: 1, NombreRol: "administrador", Nivel: 4, Activo: true},
			},
			"inactivo": {
				ID: 2, Username: "inactivo", Email: "inactivo@test.com",
				Password:  hashPassword(t, "secret"),
				FirstName: "In", LastName: "Activo", Legajo: "I0001",
				Department: "Test", Activo: false,
				Rol: Rol{ID: 4, NombreRol: "produccion", Nivel: 1, Activo: true},
			},
		},
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
	m.createID++
	u.ID = m.createID
	m.users[u.Username] = u
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
