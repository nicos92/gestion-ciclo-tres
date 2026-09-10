# Plan detallado — Fase 3 (F3): Tarimas alta/edición

## Objetivo
Implementar alta y edición de tarimas con parseo del código de barras en dominio (con tests), validaciones server-side, permisos por nivel (RBAC), y alertas inline vía htmx. El usuario puede crear tarimas (nivel ≥1) y editarlas (nivel ≥2) con auto-relleno desde el código de barras.

## Criterio de aceptación (def-done)
- `go vet ./...` sin errores.
- `go build ./...` y `CGO_ENABLED=0 go build ./...` OK.
- `go test ./...` OK (dominio: parseo barcode, validación, creación/edición; repos: CRUD en SQLite temporal).
- `go run .` levanta; login → `/tarimas` → botón "Nueva Tarima" funciona.
- `GET /tarimas/nueva` muestra formulario con focus en código de barras.
- `POST /tarimas` crea tarima, responde fragmento htmx con alerta success + reset form.
- Duplicado de código de barras → alerta error inline (sin redirect).
- `GET /tarimas/editar/{id}` (nivel ≥2) muestra formulario pre-llenado.
- `POST /tarimas/actualizar/{id}` actualiza tarima, responde fragmento htmx con alerta success.
- `GET /tarimas/nueva` sin sesión → redirect a `/login`.
- `GET /tarimas/nueva` con nivel 1 (produccion) → permitido.
- `GET /tarimas/editar/{id}` con nivel 1 → redirect (sin permiso).

## 1. Formato de código de barras (30 dígitos)
```
Pos  0:      "0"          (1 dígito, fijo)
Pos  1-6:    producto     (6 dígitos)
Pos  7-12:   tarima       (6 dígitos)
Pos 13-16:   "9998"       (4 dígitos, marcador de tarima)
Pos 17:      conservación (1 dígito)
Pos 18-20:   usuario      (3 dígitos)
Pos 21-23:   cajas        (3 dígitos)
Pos 24-29:   peso         (6 dígitos: 4 enteros + 2 decimales)
```
Este formato reemplaza al de `buildBarcode()` de F0 (usuario 2 dígitos + cajas 5 dígitos) e incorpora el campo `conservacion`.

## 2. Base de datos (migraciones unidas)
- `migrations/0001_init.sql`: se agrega `conservacion TEXT DEFAULT ''` directamente en `CREATE TABLE tarimas` y `CREATE TABLE historial_tarimas` (sin migración ALTER separada; se reemplaza el archivo con la columna incorporada).
- `migrations/0003_vista_tarimas.sql`: la vista `vista_tarimas_con_legajo` incluye `COALESCE(t.conservacion, '') AS conservacion`.
- `seed.go`: `buildBarcode(numeroProducto, numeroTarima, numeroUsuario, conservacion string, cajas int, peso float64)` genera el código con el nuevo formato; `seedTarimas` y `insertTarima` incluyen `conservacion`.

## 3. Dominio

### 3.1 `internal/tarima/tarima.go`
- Campo nuevo `Conservacion string` en `Tarima`.
- Sentinel errors:
  - `ErrBarcodeLongitud`, `ErrBarcodePrefix`, `ErrBarcodeMarker`
  - `ErrCodigoBarrasRequerido`, `ErrNumeroTarimaRequerido`
  - `ErrNumeroVentaFormato`, `ErrNumeroVentaMax`
  - `ErrNumeroProductoMax` (6), `ErrNumeroTarimaMax` (6), `ErrNumeroUsuarioMax` (3)
  - `ErrCantidadCajasRango` (1-999), `ErrPesoRango` (0-9999.99)
  - `ErrCodigoBarrasDuplicado`, `ErrTarimaNoEncontrada`
- `ValidateTarima(t *Tarima) error`: valida campos con las reglas del PHP (regex venta `^\d{2}-\d{6}$`, longitudes máximas, rangos).

### 3.2 `internal/tarima/barcode.go` (nuevo)
- `BarcodeData{NumeroProducto, NumeroTarima, NumeroUsuario, Conservacion, CantidadCajas, Peso}`.
- `ParseBarcode(raw string) (BarcodeData, error)`: valida longitud 30, prefijo `'0'`, marcador `'9998'` en `raw[13:17]`, extrae según las posiciones de la sección 1, convierte peso de centavos (`raw[24:30] / 100`).

### 3.3 `internal/tarima/repository.go`
Interfaz ampliada:
```go
type TarimaRepository interface {
    ListToday(ctx context.Context, limit int) ([]Tarima, error)
    ListAll(ctx context.Context, limit int) ([]Tarima, error)
    ListFiltered(ctx context.Context, filters FiltrosTarima, limit int) ([]Tarima, error)
    CountToday(ctx context.Context) (int, error)
    Create(ctx context.Context, t *Tarima) (int64, error)
    GetByID(ctx context.Context, id int64) (*Tarima, error)
    Update(ctx context.Context, t *Tarima) error
}
```

### 3.4 `internal/tarima/service.go`
- `Create(ctx, t)`: trim de campos → si `len(CodigoBarras)==30`, `ParseBarcode` auto-rellena campos vacíos (producto, tarima, usuario, conservacion, cajas=0, peso=0) → `ValidateTarima` → `repo.Create`. Error con `UNIQUE constraint failed` → `ErrCodigoBarrasDuplicado`.
- `GetByID(ctx, id)`: delega; `sql.ErrNoRows` → `ErrTarimaNoEncontrada`.
- `Update(ctx, t)`: trim → `ValidateTarima` → `repo.Update`; UNIQUE → `ErrCodigoBarrasDuplicado`.

### 3.5 Tests de dominio
- `barcode_test.go`: parseo válido, peso con decimales, longitud incorrecta, prefijo incorrecto, marcador incorrecto, ceros, peso máximo, y validación completa/requeridos/formato/rangos/longitudes.
- `service_test.go`: mock repo extendido con `Create`/`GetByID`/`Update`; tests de Create éxito/campos requeridos/duplicado, GetByID éxito/no encontrado, Update éxito/duplicado.

## 4. Infraestructura SQLite

### 4.1 `internal/infrastructure/sqlite/tarima_repository.go`
- `tarimaSelectCols` incluye `conservacion` (después de `numero_usuario`).
- `Create`: `INSERT INTO tarimas (codigo_barras, numero_producto, numero_tarima, numero_usuario, conservacion, cantidad_cajas, peso, numero_venta, descripcion, id_usuario) VALUES (...)` → `LastInsertId()`.
- `GetByID`: `SELECT <cols> FROM vista_tarimas_con_legajo WHERE id = ?` (scan único con `sql.NullInt64` para id_usuario, parse UTC).
- `Update`: `UPDATE tarimas SET ... WHERE id = ?` (sin `updated_at`: la tabla `tarimas` no tiene esa columna); `RowsAffected()==0` → `ErrTarimaNoEncontrada`.
- `buildFilterQuery`: sin cambios (conservación no es filtro).

### 4.2 `internal/infrastructure/sqlite/tarima_repository_test.go`
- `insertTestTarima` actualizado: firma con `conservacion`, usa `buildBarcode` nuevo.
- Tests nuevos: Create éxito (verifica GetByID posterior y conservacion), Create duplicado, GetByID existe/no existe, Update éxito (verifica campos y conservacion), Update no encontrado.
- Renombrado `TestTarimaCreate_Duplicado` para evitar colisión con el test homónimo de `user_repository_test.go` (mismo package).

## 5. Delivery: handlers

### 5.1 `internal/delivery/handlers/tarima.go`
Data struct nuevo `tarimaFormData{Title, AppName, Session, Tarima *tarima.Tarima, Error, Success string, EditMode bool}`.

| Método | Path | Nivel | Comportamiento |
|--------|------|-------|----------------|
| `ShowNuevaTarima` | `GET /tarimas/nueva` | 1 | Render página completa `nueva_tarima` |
| `GuardarTarima` | `POST /tarimas` | 1 | ParseForm → `Create` → fragmento `form_tarimas` success/error |
| `ShowEditarTarima` | `GET /tarimas/editar/{id}` | 2 | `GetByID` → Render `editar_tarima` (redirect a `/tarimas` si no existe) |
| `ActualizarTarima` | `POST /tarimas/actualizar/{id}` | 2 | ParseForm → `Update` → fragmento success/error |

Detalles:
- `IDUsuario` se toma de la sesión (`session.UserID`).
- `GuardarTarima`: error `ErrCodigoBarrasDuplicado` → `Error="duplicate"`; resto → `Error="validation"`. Éxito → `Success="created"` + header `HX-Push-Url: /tarimas/nueva`.
- `ActualizarTarima`: éxito → recarga con `GetByID` y `Success="updated"`.

### 5.2 htmx (formularios)
- Form con `hx-post` (al path de alta o actualización) + `hx-target="#form-container"` + `hx-swap="outerHTML"`.
- El servidor responde el mismo partial `form_tarimas` con la alerta (success/error) cargada; no hay redirect.

## 6. Templates

| Archivo | Tipo | Contenido |
|---------|------|-----------|
| `web/templates/nueva_tarima.html` | Página completa | Card "Formulario de Nueva Tarima" + `{{template "form_tarimas" .}}` + focus autofoco en barcode + auto-close de alerta (5s) |
| `web/templates/editar_tarima.html` | Página completa | "Editar Tarima" + `{{template "form_tarimas" .}}`; alerta "Tarima no encontrada" si `Tarima == nil` |
| `web/templates/partials/form_tarimas.html` | Fragmento htmx (nuevo) | Wrapper `#form-container`, alertas success/error, form con `hx-post` condicional según `EditMode`, campos con `value` pre-llenado, botón "Guardar"/"Actualizar" |
| `web/templates/tarimas.html` | Modificación | Botón "Nueva Tarima" `href="#"` → `href="/tarimas/nueva"` (sin `title` de placeholder) |

- `numeroUsuario`: `maxlength="3"` (antes 2).
- Input `conservacion`: campo nuevo (`maxlength="1"`, icono snowflake).
- Validación HTML5: `required`, `pattern="\d{2}-\d{6}"`, `min`/`max` cajas 1-999, peso 0-9999.99.

## 7. `web/static/shared.js`
- `autoFillFromBarcode(input)`: al llegar a 30 dígitos valida `'0'` y `'9998'`, rellena producto `[1:7]`, tarima `[7:13]`, conservación `[17]`, usuario `[18:21]`, cajas `[21:24]` (parseInt), peso `[24:28].[28:30]`, setea venta con año actual + "-" y hace focus en venta.

## 8. Cambios en main.go
```go
// Tarimas — alta (nivel 1)
mux.Handle("GET /tarimas/nueva",  AuthRequired(NivelRequerido(1, ShowNuevaTarima)))
mux.Handle("POST /tarimas",       AuthRequired(NivelRequerido(1, GuardarTarima)))
// Tarimas — edición (nivel 2)
mux.Handle("GET /tarimas/editar/{id}",  AuthRequired(NivelRequerido(2, ShowEditarTarima)))
mux.Handle("POST /tarimas/actualizar/{id}", AuthRequired(NivelRequerido(2, ActualizarTarima)))
```

### RBAC por nivel (F3)
| Endpoint | Nivel mínimo | Acción |
|----------|-------------|--------|
| `GET /tarimas/nueva` | 1 | Ver formulario de alta |
| `POST /tarimas` | 1 | Crear tarima |
| `GET /tarimas/editar/{id}` | 2 | Ver formulario de edición |
| `POST /tarimas/actualizar/{id}` | 2 | Actualizar tarima |

## 9. Orden de ejecución
| # | Archivo | Acción |
|---|---------|--------|
| 1 | `migrations/0001_init.sql` | Agregar `conservacion` a `CREATE TABLE` de `tarimas` y `historial_tarimas` |
| 2 | `migrations/0003_vista_tarimas.sql` | Incluir `conservacion` en la vista |
| 3 | `internal/tarima/tarima.go` | Campo `Conservacion` + sentinel errors + `ValidateTarima` |
| 4 | `internal/tarima/barcode.go` | `BarcodeData` + `ParseBarcode` |
| 5 | `internal/tarima/barcode_test.go` | Tests de parseo y validación |
| 6 | `internal/tarima/repository.go` | Agregar `Create`, `GetByID`, `Update` |
| 7 | `internal/tarima/service.go` | Agregar `Create`, `GetByID`, `Update` (con auto-relleno y UNIQUE mapping) |
| 8 | `internal/tarima/service_test.go` | Tests con mock repo extendido |
| 9 | `internal/infrastructure/sqlite/tarima_repository.go` | Implementar `Create`, `GetByID`, `Update` + `conservacion` |
| 10 | `internal/infrastructure/sqlite/tarima_repository_test.go` | Tests CRUD + `conservacion` |
| 11 | `internal/infrastructure/sqlite/seed.go` | `buildBarcode` nuevo formato + `conservacion` en seed |
| 12 | `web/templates/partials/form_tarimas.html` | Fragmento del form con alertas inline |
| 13 | `web/templates/nueva_tarima.html` | Página completa de alta |
| 14 | `web/templates/editar_tarima.html` | Página completa de edición |
| 15 | `web/static/shared.js` | `autoFillFromBarcode` |
| 16 | `web/templates/tarimas.html` | Botón activo → `/tarimas/nueva` |
| 17 | `internal/delivery/handlers/tarima.go` | Handlers de alta/edición |
| 18 | `main.go` | Registrar rutas + `NivelRequerido` |
| 19 | Verificación | vet + build + test + smoke curl |

## 10. Verificación F3
```
go vet ./...
go build ./...
$env:CGO_ENABLED='0'; go build ./...
go test ./...
go run .

# Smoke tests:
GET /tarimas/nueva              → 302 a /login (sin sesión)
POST /login (admin/password)    → HX-Redirect /
GET /tarimas/nueva              → 200 HTML con form, focus en barcode
POST /tarimas                   → 200 fragmento HTML con alerta success (o error)
POST /tarimas (duplicado)       → 200 fragmento con alerta error "Ya existe una tarima..."
GET /tarimas/editar/{id}        → 200 HTML con form pre-llenado
POST /tarimas/actualizar/{id}   → 200 fragmento HTML con alerta success
GET /tarimas/nueva (nivel 1)    → 200 OK (produccion puede crear)
GET /tarimas/editar/{id} (nivel 1) → 302 a / (produccion no puede editar)
```

## 11. Detalles técnicos resueltos
- **Formato de barcode**: el parseo usa `[17]` conservación, `[18:21]` usuario (3 dígitos), `[21:24]` cajas (3 dígitos), `[24:30]` peso; coincide con `buildBarcode` (seed) y `autoFillFromBarcode` (JS).
- **Sin `updated_at` en tarimas**: la tabla no tiene esa columna, por lo que el `UPDATE` no la toca (a diferencia del plan original); se verifica con `RowsAffected()`.
- **Error UNIQUE**: detección por `strings.Contains(err.Error(), "UNIQUE constraint failed")`; mapeado a `ErrCodigoBarrasDuplicado` en el servicio.
- **Colisión de nombres de test**: `TestTarimaCreate_Duplicado` evita chocar con `TestCreate_Duplicado` de `user_repository_test.go` (mismo package `sqlite`).
- **Fragmentos htmx**: el POST responde el partial `form_tarimas` (no una redirección); éxito de alta setea `HX-Push-Url: /tarimas/nueva`.

## 12. Límites de F3
- Sin eliminación ni historial (F4).
- Sin columna Acciones en la tabla (F4 agrega "Editar" y "Eliminar").
- Sin dashboard ni gestión de usuarios (F5).
- Sin Docker (F6).