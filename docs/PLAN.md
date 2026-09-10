# Plan general: migración PHP → Go + htmx + SQLite

## Objetivo
Portar la aplicación PHP "Gestión de Tarimas" (`appphp/`) a Go con `html/template` + htmx + SQLite, usando arquitectura basada en dominios y lista para desplegar en Docker (Linux).

## Stack
- Go 1.27 estándar: `net/http`, `html/template`, `database/sql`.
- `modernc.org/sqlite` (puro Go, `CGO_ENABLED=0` → igual en Windows, Linux y contenedor).
- `golang.org/x/crypto` (bcrypt; compatibilidad `$2y$` → `$2a$` al verificar hashes existentes).
- htmx por CDN (Bootstrap + FontAwesome también por CDN, como en PHP).

## Módulo y configuración
- Módulo Go: `gestion-ciclo-tres` → binario/imagen/servicio Docker `gestion-ciclo-tres`.
- Config por variables de entorno (`internal/config`):
  - `PORT` (default `8080`)
  - `DB_PATH` (default local `./data/gestiontarimas.db`; en contenedor `/data/gestiontarimas.db`)
  - `TZ` (default `America/Argentina/Buenos_Aires`; afecta el filtro "hoy")
  - `APP_NAME` (default `Gestión de Tarimas`)

## Arquitectura (bounded contexts)
```
main.go                        composition root: config + DI + http.Server + graceful shutdown (SIGINT/SIGTERM)
internal/
  config/                      configuración por env
  identity/                    dominio auth/usuarios: Usuario, Rol (nivel), AuthService, UserRepository (interfaz)
  tarima/                      dominio inventario: Tarima, CodigoBarras (value object), FiltrosTarima,
                               TarimaService (crear/editar/buscar/eliminar→historial), TarimaRepository (interfaz)
  infrastructure/sqlite/       repos concretos, conexión (PRAGMAs), migraciones + seed embebidos
  delivery/
    render/                    template.ParseFS (go:embed) + helpers de formato
    middleware/                sesión server-side (cookie + store en memoria), auth_required, nivel_requerido
    handlers/                  auth, tarimas, usuarios, dashboard, healthz (fragmentos htmx)
web/
  templates/                   layout + páginas + parciales htmx (go:embed)
  static/shared.js             embebido + http.FileServer (máscaras de input)
migrations/0001_init.sql
Dockerfile · .dockerignore · docker-compose.yml
```
Reglas de negocio en el dominio; repos como interfaces (testeable con SQLite temporal); `delivery` solo traduce HTTP ↔ dominio.

## Base de datos (SQLite)
- Tablas: `roles`, `usuarios`, `tarimas`, `historial_tarimas`, `schema_migrations`.
- Se elimina la matriz `permisos`/`roles_permisos` de MySQL → RBAC por nivel.
- `roles(id, nombre_rol, nivel, descripcion, activo)`.
- `ON UPDATE CURRENT_TIMESTAMP` (MySQL) → se setea `updated_at` en el UPDATE.
- `fecha DATE DEFAULT (CURRENT_DATE())` → `(date('now'))`.
- Vista `vista_tarimas_con_legajo` y procedimiento `FiltrarTarimas` → JOIN y query dinámico en Go.
- Trigger `after_tarima_delete` → transacción explícita en el servicio: INSERT historial + DELETE.
- `PRAGMA journal_mode=WAL`, `busy_timeout`, `foreign_keys=ON`.
- Seed solo con datos de ejemplo; la migración de datos reales queda a cargo del usuario.

## RBAC por rango
Niveles: `produccion`(1) < `supervisor`(2) < `jefe_produccion`(3) < `administrador`(4).
| Acción | Nivel mínimo |
|---|---|
| Ver dashboard / lista / crear tarima | 1 |
| Eliminar tarima (+ historial) | 2 (supervisor) |
| Editar tarima | 3 |
| Registrar / editar usuarios | 4 |

## Reglas de negocio a portar al dominio
- Parseo del código de barras de 30 dígitos (producto, tarima, usuario, cajas, peso, venta) con fallbacks.
- Validaciones de alta/edición (formato de venta `XX-XXXXXX`, límites de longitud, duplicados de código de barras).
- Filtros del listado (producto, tarima, usuario, venta, fecha, legajo, nombre, cajas mín, peso mín) y modo "hoy" / "all=1".

## htmx (vs. redirect + query params de PHP)
- Filtros: `hx-get`, `hx-target` sobre la tabla, `hx-swap="outerHTML"`, `hx-push-url` (mantiene URL imprimible/compartible).
- Alta/edición: fragmento con alerta inline + reset y re-focus del input; errores de duplicado inline.
- Eliminar (nuevo): `hx-delete`, `hx-confirm`, `hx-target="closest tr"`, `hx-swap="delete"`, servidor responde `200` vacío (el `204` desactiva el swap en htmx).

## Fases (hitos)
- **F0** Esqueleto + DB + base Docker: config por env, `/healthz`, graceful shutdown, SQLite con PRAGMAs, migración + seed, roles con niveles.
- **F1** Auth: login/register (admin)/logout, bcrypt ($2y→$2a), sesión server-side por cookie, layout + htmx.
- **F2** Tarimas listado: vista hoy / `all=1` / filtros dinámicos con htmx.
- **F3** Tarimas alta/edición: parseo del código de barras en dominio (con tests), permisos por nivel, alertas inline.
- **F4** Eliminar + historial: transacción + permiso supervisor + `hx-delete`.
- **F5** Usuarios + dashboard: listar/editar (admin), stats con formateo de miles.
- **F6** Docker: Dockerfile multi-stage (golang:1.27-alpine + `CGO_ENABLED=0` → alpine no-root), `.dockerignore`, compose con volumen persistente y `TZ`.

## Verificación en cada fase
`go vet ./...` · `go build ./...` · `go test ./...` (dominio: parseo, filtros, permisos, eliminación+historial; repos: CRUD en SQLite temporal) · smoke test por endpoint vía curl.

## Notas
- `appphp/` se conserva como referencia y se excluye del build Docker.
- Sesión en memoria: se pierde al reiniciar (aceptable para una instancia única en Docker).
- El demo "todo" existente en la raíz se reemplaza por esta aplicación.