# Plan detallado — Fase 1 (F1): Auth + layout + htmx

## Objetivo
Implementar autenticación completa (login, register solo admin, logout) con sesión server-side en memoria, layout Bootstrap+htmx, y los primeros templates renderizados desde Go. El usuario puede loguearse, ver una página protegida, y cerrar sesión.

## Criterio de aceptación (def-done)
- `go vet ./...` sin errores.
- `go build ./...` y `CGO_ENABLED=0 go build ./...` OK.
- `go test ./...` OK (dominio + repos en SQLite temporal).
- `go run .` levanta; `GET /login` muestra el form.
- Login con credenciales correctas → redirect a `/` (dashboard placeholder) con usuario en sesión.
- Login con credenciales incorrectas → redirect a `/login?error=invalid_credentials`.
- Login con usuario inactivo → redirect a `/login?error=inactive_user`.
- `GET /register` (con admin logueado) muestra form de registro.
- `GET /register` sin sesión → redirect a `/login`.
- `GET /register` con nivel < 4 → redirect a `/`.
- Registro exitoso → redirect a `/register?success=true`.
- `POST /logout` → destruye sesión → redirect a `/login`.
- Cookies: `session_id`, HttpOnly, SameSite=Lax.

## 1. Paquetes nuevos

### 1.1 `internal/identity/usuario.go` — domain
- Structs: `Rol`, `Usuario`
- Errores: `ErrUsuarioNoEncontrado`, `ErrCredencialesInvalidas`, `ErrUsuarioInactivo`, `ErrUsernameExiste`, `ErrEmailExiste`

### 1.2 `internal/identity/repository.go` — interfaz
- `UserRepository`: `GetByUsername`, `Create`, `ExistsByUsernameOrEmail`

### 1.3 `internal/identity/service.go` — AuthService
- `Login(ctx, username, password) (*Usuario, error)`
- `Register(ctx, *Usuario, password) (int64, error)`
- bcrypt `$2a$`/`$2y$` manejado nativamente por `bcrypt.CompareHashAndPassword`

### 1.4 `internal/infrastructure/sqlite/user_repository.go`
- Implementación concreta de `UserRepository` con queries SQL

### 1.5 `internal/infrastructure/sqlite/user_repository_test.go`
- Tests con DB en `t.TempDir()`: GetByUsername, Create, ExistsByUsernameOrEmail

### 1.6 `internal/identity/service_test.go`
- Tests con SQLite temporal: Login OK/fallido/inactivo/no existe, Register OK/duplicado

## 2. Sesión server-side

### 2.1 `internal/delivery/middleware/session.go`
- `SessionData{UserID, Username, Nivel, NombreRol}`
- `SessionStore` con `sync.RWMutex` + `map[string]*SessionData`
- Cookie: `session_id`, HttpOnly, SameSite=Lax, Path=/
- Session ID: `crypto/rand` → 32 bytes hex

### 2.2 `internal/delivery/middleware/auth.go`
- `AuthRequired(store)` → middleware que verifica sesión
- `NivelRequerido(store, nivelMinimo)` → middleware que verifica nivel
- `SessionFromContext(ctx)` → helper para extraer sesión del context

## 3. Render templates

### 3.1 `internal/delivery/render/render.go`
- `Renderer` con `template.ParseFS`
- Funciones template: `formatDate`, `formatMoney`, `nl2br`, `toFloat`
- Embed de `web/templates/**` en `main.go`

## 4. Templates web

### 4.1 `web/templates/layout.html`
- Bootstrap 5.3, FontAwesome 6.4, htmx 2.0.4 por CDN
- CSS del PHP main.php

### 4.2 `web/templates/login.html`
- Form Bootstrap, `hx-post="/login" hx-swap="none"`
- Alertas por query param `?error=`

### 4.3 `web/templates/register.html`
- Form completo (nombre, apellido, email, username, legajo, password, dept, rol, activo)
- `hx-post="/register" hx-swap="none"`

### 4.4 `web/templates/partials/header.html`
- Barra con usuario + logout o solo título

### 4.5 `web/static/shared.js`
- `togglePassword`, `checkPasswordMatch`

## 5. Handlers auth

### 5.1 `internal/delivery/handlers/auth.go`
- `ShowLogin` (GET /login) → render form o redirect si logueado
- `Login` (POST /login) → auth.Login → HX-Redirect
- `ShowRegister` (GET /register) → requiere nivel 4 → render form
- `Register` (POST /register) → auth.Register → HX-Redirect
- `Logout` (POST /logout) → destruir sesión → HX-Redirect /login

### 5.2 Header HX-Redirect
- Server devuelve `HX-Redirect: <url>` → htmx redirige automáticamente

## 6. Cambios en main.go
- Embed `web/templates` y `web/static`
- Crear SessionStore, Renderer, UserRepo, AuthHandler
- Registrar rutas auth (públicas y protegidas)
- Static files en `/static/`
- Dashboard placeholder protegido

## 7. Orden de ejecución

| # | Archivo |
|---|---|
| 1 | `docs/PLAN-F1.md` |
| 2 | `internal/identity/usuario.go` |
| 3 | `internal/identity/repository.go` |
| 4 | `internal/infrastructure/sqlite/user_repository.go` |
| 5 | `internal/infrastructure/sqlite/user_repository_test.go` |
| 6 | `internal/identity/service.go` |
| 7 | `internal/identity/service_test.go` |
| 8 | `internal/delivery/render/render.go` |
| 9 | `web/templates/layout.html` |
| 10 | `web/templates/partials/header.html` |
| 11 | `web/templates/login.html` |
| 12 | `web/templates/register.html` |
| 13 | `web/static/shared.js` |
| 14 | `internal/delivery/middleware/session.go` |
| 15 | `internal/delivery/middleware/auth.go` |
| 16 | `internal/delivery/handlers/auth.go` |
| 17 | `main.go` (modificar) |

## 8. Verificación F1
```
go vet ./...
go build ./...
CGO_ENABLED=0 go build ./...
go test ./...
go run .
curl http://localhost:8080/login           → 200 form login
curl -X POST http://localhost:8080/login -d "username=admin&password=password" → HX-Redirect /
curl http://localhost:8080/register        → 302 redirect a /login (sin sesión)
```

## 9. RBAC por nivel

| Endpoint | Nivel mínimo |
|---|---|
| `GET/POST /login` | 0 (público) |
| `POST /logout` | 1 |
| `GET /` (dashboard) | 1 |
| `GET/POST /register` | 4 (admin) |
| `GET /healthz` | 0 (público) |
