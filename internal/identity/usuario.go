package identity

import (
	"errors"
	"time"
)

var (
	ErrUsuarioNoEncontrado    = errors.New("usuario no encontrado")
	ErrCredencialesInvalidas  = errors.New("credenciales inválidas")
	ErrUsuarioInactivo        = errors.New("usuario inactivo")
	ErrUsernameExiste         = errors.New("el nombre de usuario ya existe")
	ErrEmailExiste            = errors.New("el email ya está registrado")
	ErrCamposRequeridos       = errors.New("todos los campos son obligatorios")
	ErrPasswordNoCoinciden    = errors.New("las contraseñas no coinciden")
	ErrEmailInvalido          = errors.New("email inválido")
)

type Rol struct {
	ID          int64
	NombreRol   string
	Nivel       int
	Descripcion string
	Activo      bool
}

type Usuario struct {
	ID         int64
	Username   string
	Email      string
	Password   string
	FirstName  string
	LastName   string
	Legajo     string
	Department string
	Rol        Rol
	Activo     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
