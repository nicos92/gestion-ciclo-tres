# Plan detallado — Fase 2 (F2): Tarimas listado: vista hoy / all / filtros dinámicos con htmx

## Objetivo
Implementar el listado de tarimas con tres modos (hoy, todos, filtrados), filtros dinámicos vía htmx, y la vista SQL `vista_tarimas_con_legajo`. El usuario puede ver las tarimas del día, buscar todas, y filtrar por 9 parámetros sin recarga de página completa.

## Criterio de aceptación (def-done)
- `go vet ./...` sin errores.
- `go build ./...` y `CGO_ENABLED=0 go build ./...` OK.
- `go test ./...` OK (dominio: filtros; repos: CRUD tarimas en SQLite temporal).
- `go run .` levanta; login → `/tarimas` muestra tabla con tarimas del día.
- Sin filtros ni `?all=1`: muestra solo tarimas de hoy (máx 1000).
- `GET /tarimas?all=1`: muestra últimas 1000 tarimas sin filtro de fecha.
- Filtros activos (botón "Filtrar"): resultados filtrados (máx 1000), buscando en todas las fechas.
- htmx: filtros se aplican sin recarga de página (`hx-get`, `hx-target`, `hx-swap="outerHTML"`), con `HX-Push-Url` a la URL canónica.
- Badges: "Tarimas ingresadas hoy: N" y "Tarimas mostradas: N" siempre visibles.
- `GET /tarimas` sin sesión → redirect a `/login`.

## 1. Paquetes nuevos

### 1.1 `internal/tarima/tarima.go` — domain
- `Tarima` struct: `ID`, `CodigoBarras`, `NumeroProducto`, `NumeroTarima`, `NumeroUsuario`, `CantidadCajas`, `Peso`, `NumeroVenta`, `Descripcion`, `IDUsuario` (*int64), `FechaRegistro`, `Fecha`, `Legajo`, `NombreUsuario` (estos últimos dos vienen del JOIN).
- `FiltrosTarima` struct: los 9 parámetros del PHP (`NumeroProducto`, `NumeroTarima`, `NumeroUsuario`, `NumeroVenta`, `FechaRegistro`, `Legajo`, `NombreUsuario`, `CantidadCajasMin` *int, `PesoMin` *float64).
- `HasFilters() bool`: true si algún campo está poblado (incluidos los punteros).

### 1.2 `internal/tarima/repository.go` — interfaz
```go
type TarimaRepository interface {
    ListToday(ctx context.Context, limit int) ([]Tarima, error)
    ListAll(ctx context.Context, limit int) ([]Tarima, error)
    ListFiltered(ctx context.Context, filters FiltrosTarima, limit int) ([]Tarima, error)
    CountToday(ctx context.Context) (int, error)
}
```

### 1.3 `internal/tarima/service.go` — lógica de modos
- Constantes `DefaultLimit = 1000`, `MaxLimit = 10000`.
- `List(ctx, filters, showAll, limit)`:
  - `limit <= 0` → default (1000); `limit > MaxLimit` → cap (10000).
  - `filters.HasFilters()` → `ListFiltered` (prioridad sobre all).
  - `showAll` → `ListAll`.
  - si no → `ListToday`.
- `CountToday(ctx)` → delega en repo.

### 1.4 `internal/tarima/service_test.go`
- Mock `TarimaRepository` que registra qué método se llamó.
- Tests: default→today, all→ListAll, filtros→ListFiltered, precedencia de filtros sobre all, caps de límite, `HasFilters` (tabla de casos).

## 2. Infraestructura SQLite

### 2.1 Migración `migrations/0003_vista_tarimas.sql`
```sql
CREATE VIEW IF NOT EXISTS vista_tarimas_con_legajo AS
SELECT
    t.id, t.codigo_barras, t.numero_producto, t.numero_tarima, t.numero_usuario,
    t.cantidad_cajas, t.peso, t.numero_venta,
    COALESCE(t.descripcion, '') AS descripcion,
    t.id_usuario, t.fecha_registro, t.fecha,
    COALESCE(u.legajo, '') AS legajo,
    COALESCE(u.first_name || ' ' || u.last_name, '') AS nombre_usuario
FROM tarimas t
LEFT JOIN usuarios u ON t.id_usuario = u.id;
```
Notas:
- SQLite no tiene `CONCAT` → se usa `||`.
- `COALESCE` para `descripcion`, `legajo` y `nombre_usuario` (evita `NULL` → error de scan).
- `id_usuario` puede ser `NULL` (ON DELETE SET NULL) → se escanea con `sql.NullInt64`.

### 2.2 `internal/infrastructure/sqlite/tarima_repository.go`
- `NewTarimaRepository(db)` → implementación concreta.
- Constante `tarimaSelectCols` con las columnas de la vista.
- `ListToday`: `WHERE date(fecha_registro, 'localtime') = date('now', 'localtime') ORDER BY fecha_registro DESC LIMIT ?`.
- `ListAll`: sin WHERE por fecha, `ORDER BY fecha_registro DESC LIMIT ?`.
- `CountToday`: `COUNT(*)` sobre la misma condición "hoy".
- `ListFiltered`: `buildFilterQuery(filters, limit)` → `strings.Builder` con WHERE condicional.
- Los timestamps del driver (RFC3339 UTC, ej. `2026-09-10T11:28:00Z`) se escanean como string y se parsean como UTC (`parseSQLTimestamp` / `parseSQLDate`) para que `formatDate` los convierta a `time.Local`.

### 2.3 `buildFilterQuery` — query dinámico (replica del stored procedure `FiltrarTarimas`)
- `SELECT ... FROM vista_tarimas_con_legajo WHERE 1=1` + cláusulas condicionales.
- LIKE parcial: `numero_producto`, `numero_tarima`, `numero_usuario`, `numero_venta`, `legajo`, `nombre_usuario` → `LIKE '%' || ? || '%'`.
- Exacto: `fecha_registro` → `date(fecha_registro, 'localtime') = ?`.
- Mínimo (>=): `cantidad_cajas_min`, `peso_min`.
- Cierra con `ORDER BY fecha_registro DESC LIMIT ?`.

### 2.4 `internal/infrastructure/sqlite/tarima_repository_test.go`
- Helper `newTestTarimaRepo` + `insertTestTarima` (inserta con fechas explícitas UTC).
- Tests: ListToday solo devuelve hoy, ListAll devuelve todo, filtros por producto/legajo(con JOIN)/cajas_min/peso_min/fecha, CountToday, límite de resultados.

## 3. Handler

### 3.1 `internal/delivery/handlers/tarima.go`
- `TarimaHandler{renderer, tarima *tarima.TarimaService, sessions}`.
- Data: `tarimasPageData{Title, AppName, Session, Tarimas, CountToday, Filters, ShowAll}`.

| Método | Path | Nivel | Respuesta |
|--------|------|-------|-----------|
| `GET /tarimas` | 1 | Página completa (layout + widgets + tabla) |
| `GET /tarimas/lista` | 1 | Solo fragmento htmx (tabla/cards) |

- `ListarTarimas`: sesión del contexto → `parseFilters(r)` → `all=1` → `loadTarimas` → render "tarimas".
- `ListarTarimasFragment`: misma carga, setea header `HX-Push-Url` con la URL canónica (`/tarimas?<filtros>`), render via `RenderPartial("tabla_tarimas")`.
- `parseFilters(r)`: lee los 9 params, parsea `cantidad_cajas_min` (int) y `peso_min` (float) solo si vienen poblados.
- `canonicalTarimasURL(filters, showAll)`: construye `/tarimas?...` con `url.Values`; devuelve `""` si no hay parámetros (sin header → no push).
- `loadTarimas`: `tarima.List(ctx, filters, showAll, DefaultLimit)` + `CountToday`.

### 3.2 htmx (UX decidido con el usuario)
- Filtros solo con el botón "Filtrar" (submit), no on-change.
- `hx-push-url` sí, en el fragmento: se implementa vía header `HX-Push-Url` del servidor (no con el atributo `hx-push-url` estático), porque la URL canónica/imprimible debe ser `/tarimas?<filtros>` y no `/tarimas/lista?...`.

## 4. Templates

### 4.1 `web/templates/tarimas.html` — página completa
- Define `{{define "styles"}}` con el CSS de impresión (del PHP): ocultar botones, A4, 8pt, tabla visible, `.print-filters` visible, `.no-print` oculto.
- Define `{{define "content"}}`:
  - Card header con gradiente: título "Inventario de Tarimas", botones "Nueva Tarima" (placeholder F3, `href="#"`), "Volver al Panel" (`/`), "Imprimir" (`window.print()`).
  - Badges: "Tarimas ingresadas hoy: N" (warning) y "Tarimas mostradas: N" (info, con `<strong id="count-mostradas">` para OOB swap).
  - Filtros colapsables (`data-bs-toggle="collapse"`), 9 inputs en 3 filas, con `value="{{.Filters.X}}"`.
  - Form `id="filtroTarimas"` con `hx-get="/tarimas/lista"`, `hx-target="#tabla-tarimas"`, `hx-swap="outerHTML"`.
  - Botones: Filtrar (submit), Limpiar Filtros (`limpiarFiltros()`), "Ver Últimas 1000 Tarimas" (`href="/tarimas?all=1"`).
  - `{{template "tabla_tarimas" .}}`.

### 4.2 `web/templates/partials/tabla_tarimas.html` — fragmento htmx
- Define `{{define "tabla_tarimas"}}`:
  - Wrapper `<div id="tabla-tarimas" class="card-body p-0">` (es el target del swap).
  - `.print-filters` (oculto en pantalla, listado de filtros activos) — dentro del wrapper para que se actualice en el swap.
  - Alerta informativa si `.ShowAll` ("Mostrando las últimas N tarimas... Ver solo las de hoy").
  - Tabla desktop `d-none d-xl-block` (10 columnas, thead gradiente) + cards mobile `d-xl-none`.
  - Empty state "No hay tarimas registradas aún."
  - OOB swap al final: `<strong id="count-mostradas" hx-swap-oob="true">{{len .Tarimas}}</strong>`.

### 4.3 Cambios en `internal/delivery/render/render.go`
- `RenderPartial(w, name, data, statusCode)`: escribe solo un fragmento (definido con `{{define}}`) sin layout.
- `New()` también registra cada partial como template standalone (bajo su nombre de archivo) para que `RenderPartial("tabla_tarimas", ...)` funcione.
- `formatDate` maneja `time.Time` → `v.In(time.Local).Format("2006-01-02 15:04:05")`.

### 4.4 `web/static/shared.js`
- `limpiarFiltros()`: vacía los 9 inputs y dispara `htmx.trigger('#filtroTarimas', 'submit')` (fallback a submit nativo si no hay htmx).

## 5. Cambios en main.go
- Import `internal/tarima`.
- DI: `sqlite.NewTarimaRepository(db)` → `tarima.NewTarimaService(repo)` → `handlers.NewTarimaHandler(renderer, svc, store)`.
- Rutas protegidas nivel 1: `GET /tarimas` → `ListarTarimas`, `GET /tarimas/lista` → `ListarTarimasFragment`.
- Dashboard placeholder → redirect a `/tarimas`.
- Se elimina el uso de `strconv` (dashboard inline) y el parámetro `appName` de `routes()`.

## 6. RBAC por nivel (F2)

| Endpoint | Nivel mínimo |
|---|---|
| `GET /tarimas` | 1 |
| `GET /tarimas/lista` | 1 |

Sin columna Acciones (entra en F3/F4 con sus permisos).

## 7. Orden de ejecución

| # | Archivo | Acción |
|---|---|---|
| 1 | `docs/PLAN-F2.md` | Crear este plan |
| 2 | `migrations/0003_vista_tarimas.sql` | Crear vista SQL |
| 3 | `internal/tarima/tarima.go` | Domain model + FiltrosTarima |
| 4 | `internal/tarima/repository.go` | Interfaz TarimaRepository |
| 5 | `internal/tarima/service.go` | TarimaService (lógica de modos) |
| 6 | `internal/tarima/service_test.go` | Tests de selección de modo |
| 7 | `internal/infrastructure/sqlite/tarima_repository.go` | Repo SQLite + buildFilterQuery |
| 8 | `internal/infrastructure/sqlite/tarima_repository_test.go` | Tests CRUD en SQLite temporal |
| 9 | `internal/delivery/render/render.go` | Agregar RenderPartial + formatDate time.Time |
| 10 | `web/templates/partials/tabla_tarimas.html` | Fragmento de tabla/cards (+ OOB badge) |
| 11 | `web/templates/tarimas.html` | Página completa con filtros htmx + print CSS |
| 12 | `internal/delivery/handlers/tarima.go` | TarimaHandler (Listar + Fragment) |
| 13 | `web/static/shared.js` | Agregar limpiarFiltros() |
| 14 | `main.go` | Registrar rutas + DI + redirect dashboard |
| 15 | Verificación | vet + build + test + smoke curl |

## 8. Verificación F2
```
go vet ./...
go build ./...
$env:CGO_ENABLED='0'; go build ./...
go test ./...
go run .
# Smoke tests:
GET /tarimas                   → 302 a /login (sin sesión)
POST /login (admin/password)   → HX-Redirect /
GET /tarimas                   → 200 HTML con tabla (tarimas de hoy)
GET /tarimas?all=1             → 200 todas las tarimas, alerta "todas las fechas"
GET /tarimas/lista             → 200 fragmento HTML (sin HX-Push-Url)
GET /tarimas/lista?numero_producto=880&cantidad_cajas_min=40 → 200 fragmento filtrado
                                (45 presente, 30 ausente) + HX-Push-Url: /tarimas?...
GET /tarimas/lista?all=1       → 200 fragmento + HX-Push-Url: /tarimas?all=1
```

## 9. Límites de F2
- Sin alta/edición de tarimas (F3).
- Sin eliminación ni historial (F4).
- Sin columna Acciones en la tabla (F3/F4).
- Sin barcode parsing (F3).
- Sin paginación (mismo comportamiento que PHP: límite 1000, cap 10000).
- Botón "Nueva Tarima" es placeholder (`href="#"`).

## 10. Detalles técnicos resueltos durante la implementación
- **Timestamps UTC:** `CURRENT_TIMESTAMP` guarda UTC y el driver devuelve RFC3339 (`...Z`). Se escanean como string y se parsean en UTC; `formatDate` convierte a `time.Local` (TZ de config) → la fecha/hora mostrada es local (ej. UTC 11:28 → 08:28 Argentina).
- **"Hoy" local:** `date(fecha_registro, 'localtime') = date('now', 'localtime')` para que el filtro "hoy" use la TZ configurada como el `CURDATE()` de MySQL.
- **NULL handling:** `descripcion`, `legajo`, `nombre_usuario` con `COALESCE` en la vista; `id_usuario` con `sql.NullInt64` en el scan.
- **Cap de límite:** `min(limit, 10000)` (no reemplazo por default): `limit <= 0 → 1000`, `limit > 10000 → 10000`.
- **HX-Push-Url:** header del servidor en el fragmento con la URL canónica `/tarimas?<filtros>` (compartible/imprimible); sin headers cuando no hay filtros.

## 11. Riesgos a vigilar
- La vista `CREATE VIEW IF NOT EXISTS` requiere SQLite ≥ 3.39 (cubierto por modernc).
- SQLite no tiene `CONCAT` → concatenación con `||`.
- El fecha/hora local depende de `TZ` de config (una instancia/NIC Argentina); sesiones en memoria se pierden al reiniciar (aceptado por plan).