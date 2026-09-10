# Plan detallado — Fase 5 (F5): Usuarios + Dashboard

## Objetivo
Implementar la gestión de usuarios (listar, editar) con control de acceso admin (nivel 4), el dashboard con estadísticas (total tarimas, total usuarios) con formateo de miles, y actualizar la navegación del header.

## Criterio de aceptación (def-done)
- `go vet ./...`, `go build ./...`, `go test ./...` sin errores.
- `GET /dashboard` muestra panel con 4 cards, estadísticas con formato de miles.
- `GET /usuarios` (nivel 4) muestra tabla de usuarios.
- `GET /usuarios/editar/{id}` (nivel 4) muestra form con campos pre-cargados.
- `POST /usuarios/actualizar/{id}` (nivel 4) actualiza usuario (con password opcional).
- RBAC: nivel 1 ve dashboard, nivel 4 gestiona usuarios.
- Header con navbar: Panel, Tarimas, Usuarios (si admin).

## 1. Dominio `internal/identity/`

### 1.1 `repository.go` — Interfaz extendida
```go
type UserRepository interface {
    GetByUsername(ctx context.Context, username string) (*Usuario, error)
    Create(ctx context.Context, u *Usuario) (int64, error)
    ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
    GetByID(ctx context.Context, id int64) (*Usuario, error)
    ListAll(ctx context.Context) ([]Usuario, error)
    Update(ctx context.Context, u *Usuario) error
    UpdatePassword(ctx context.Context, id int64, password string) error
    CountAll(ctx context.Context) (int, error)
    ExistsByUsernameOrEmailExcluding(ctx context.Context, username, email string, excludeID int64) (bool, error)
}
```

### 1.2 `service.go` — Métodos nuevos
- `GetByID`, `ListAll`, `Update` (con validaciones + password opcional), `UpdatePassword`, `CountAll`.

### 1.3 `service_test.go` — Tests con mock (~10 tests)

## 2. Infraestructura SQLite `internal/infrastructure/sqlite/`

### 2.1 `user_repository.go` — Implementaciones
- `GetByID`, `ListAll`, `Update`, `UpdatePassword`, `CountAll`, `ExistsByUsernameOrEmailExcluding`.

### 2.2 `user_repository_test.go` — Tests de integración (~10 tests)

### 2.3 Extensión a `internal/tarima/`
- `CountAll` en repository.go, service.go, tarima_repository.go.

## 3. Handler `internal/delivery/handlers/usuario.go` (nuevo)
- Dashboard, ListarUsuarios, ShowEditarUsuario, ActualizarUsuario.

## 4. Rutas `main.go`
- Dashboard (nivel 1), Usuarios CRUD (nivel 4), raíz → /dashboard.

## 5. Templates
- `dashboard.html`, `usuarios.html`, `editar_usuario.html` (nuevos).
- `header.html` (navbar), `tarimas.html` (link update).

## 6. Render helper
- `formatNumber` para miles con punto.

## 7. JavaScript
- Adaptar `checkPasswordMatch` para `newPassword`.

## 8. Verificación
`go vet`, `go build`, `go test`, smoke test.
