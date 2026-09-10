package identity

import (
	"context"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo UserRepository
}

func NewAuthService(repo UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*Usuario, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, ErrCamposRequeridos
	}

	u, err := s.repo.GetByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return nil, ErrCredencialesInvalidas
	}

	if !u.Activo {
		return nil, ErrUsuarioInactivo
	}

	return u, nil
}

func (s *AuthService) Register(ctx context.Context, u *Usuario, password string) (int64, error) {
	u.Username = strings.TrimSpace(u.Username)
	u.Email = strings.TrimSpace(u.Email)
	u.FirstName = strings.TrimSpace(u.FirstName)
	u.LastName = strings.TrimSpace(u.LastName)
	u.Legajo = strings.TrimSpace(u.Legajo)
	u.Department = strings.TrimSpace(u.Department)
	password = strings.TrimSpace(password)

	if u.Username == "" || u.Email == "" || u.FirstName == "" || u.LastName == "" ||
		u.Legajo == "" || u.Department == "" || password == "" {
		return 0, ErrCamposRequeridos
	}

	if _, err := mail.ParseAddress(u.Email); err != nil {
		return 0, ErrEmailInvalido
	}

	exists, err := s.repo.ExistsByUsernameOrEmail(ctx, u.Username, u.Email)
	if err != nil {
		return 0, err
	}
	if exists {
		existUser, errUser := s.repo.GetByUsername(ctx, u.Username)
		if errUser == nil && existUser != nil {
			return 0, ErrUsernameExiste
		}
		return 0, ErrEmailExiste
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	u.Password = string(hash)

	return s.repo.Create(ctx, u)
}
