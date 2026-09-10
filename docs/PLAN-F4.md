# Plan detallado — Fase 4 (F4): Eliminar tarimas + historial

## Objetivo
Implementar la eliminación de tarimas con registro en `historial_tarimas`, transacción atómica (INSERT historial + DELETE tarima), permisos por nivel (solo supervisor+), y la interacción htmx (`hx-delete` con modal Bootstrap de confirmación).

## Criterio de aceptación (def-done)
- `go vet ./...` sin errores.
- `go build ./...` OK.
- `go test ./...` OK (dominio: Delete con mock; repos: Delete + historial en SQLite temporal).
- `go run .` levanta; login → `/tarimas` muestra columna "Acciones" solo si nivel >= 2.
- Click en botón trash → modal Bootstrap de confirmación ("¿Estás seguro...?").
- Confirmar → tarima eliminada de la tabla (htmx swap delete), 200 OK.
- `historial_tarimas` tiene un registro con todos los campos de la tarima eliminada.
- Cancelar → no se elimina nada.
- `DELETE /tarimas/{id}` sin sesión → redirect a `/login`.
- `DELETE /tarimas/{id}` con nivel 1 → redirect a `/` (no puede eliminar).
- `DELETE /tarimas/{id}` inexistente → 404.

## 1. Dominio `internal/tarima/`

### 1.1 `tarima.go` — Sin cambios
Se reutiliza `ErrTarimaNoEncontrada` existente.

### 1.2 `repository.go` — Interfaz extendida
```go
type TarimaRepository interface {
    // ... los 7 métodos existentes ...
    GetByIDRaw(ctx context.Context, id int64) (*Tarima, error)  // SELECT directo de tabla tarimas
    Delete(ctx context.Context, id int64) (*Tarima, error)       // Transacción: copia a historial + DELETE
}
```
- `GetByIDRaw`: consulta la tabla `tarimas` directamente (no la vista `vista_tarimas_con_legajo`), útil para obtener datos crudos sin JOIN con usuarios.
- `Delete`: encapsula la transacción completa en el repo (no en el servicio), garantizando atomicidad.

### 1.3 `service.go` — Método Delete
```go
func (s *TarimaService) Delete(ctx context.Context, id int64) (*Tarima, error)
```
Flujo:
1. `GetByIDRaw(ctx, id)` → si `sql.ErrNoRows` retorna `ErrTarimaNoEncontrada`.
2. `repo.Delete(ctx, id)` → retorna la tarima copiada al historial.

### 1.4 `service_test.go` — Tests con mock
- Mock actualizado con `deleteCalled`, `deleteErr` y métodos `GetByIDRaw` / `Delete`.
- `TestDelete_Success`: verifica que se llama a `repo.Delete` y retorna la tarima.
- `TestDelete_NotFound`: `GetByIDRaw` retorna `sql.ErrNoRows` → `ErrTarimaNoEncontrada`, `repo.Delete` no se ejecuta.

## 2. Infraestructura SQLite `internal/infrastructure/sqlite/`

### 2.1 `tarima_repository.go` — GetByIDRaw + Delete

**`GetByIDRaw`**: SELECT directo de tabla `tarimas` (no la vista):
```sql
SELECT id, codigo_barras, numero_producto, numero_tarima, numero_usuario,
       conservacion, cantidad_cajas, peso, numero_venta, descripcion, id_usuario,
       fecha_registro, fecha
FROM tarimas WHERE id = ?
```
Constante `tarimaRawCols` con las columnas de la tabla base.

**`Delete`** — Transacción atómica:
```
BEGIN
  SELECT → obtener todos los campos de la tarima a eliminar
  INSERT INTO historial_tarimas (id_tarima_eliminada, todos los campos...)
  DELETE FROM tarimas WHERE id = ? (verificar RowsAffected == 1)
COMMIT
```
- `defer tx.Rollback()` para rollback automático si falla cualquier paso.
- Retorna la tarima copiada al historial.

### 2.2 `tarima_repository_test.go` — Tests de integración
- `TestGetByIDRaw_Exists`: crear tarima, verificar que `GetByIDRaw` la retorna.
- `TestGetByIDRaw_NotFound`: ID inexistente, error.
- `TestDelete_Success`: crear tarima, eliminarla, verificar tabla `tarimas` vacía y `historial_tarimas` con 1 registro.
- `TestDelete_VerificaCamposHistorial`: todos los campos se copian correctamente (codigo_barras, producto, tarima, cajas, peso, venta, descripcion, id_usuario).
- `TestDelete_NotFound`: ID inexistente, no inserta en historial.

## 3. Handler `internal/delivery/handlers/tarima.go`

### 3.1 Handler EliminarTarima
```go
func (h *TarimaHandler) EliminarTarima(w http.ResponseWriter, r *http.Request)
```
- Parsea `r.PathValue("id")` a `int64`.
- Llama `h.tarima.Delete(r.Context(), id)`.
- Respuestas:
  - Éxito: `w.WriteHeader(http.StatusOK)` (body vacío; htmx hace `hx-swap="delete"`).
  - `ErrTarimaNoEncontrada`: 404.
  - Otro error: 500.

### 3.2 Nota sobre 200 vs 204
El PLAN.md indica responder `200` vacío (no `204`) porque `204` desactiva el swap en htmx.

## 4. Ruta `main.go`

```go
mux.Handle("DELETE /tarimas/{id}",
    middleware.AuthRequired(store)(
        middleware.NivelRequerido(store, 2)(  // nivel 2 = supervisor
            http.HandlerFunc(tarimaHandler.EliminarTarima))))
```

## 5. RBAC por nivel (F4)

| Endpoint | Nivel mínimo |
|---|---|
| `DELETE /tarimas/{id}` | 2 (supervisor) |

## 6. Templates

### 6.1 `web/templates/partials/tabla_tarimas.html`
- Nuevo `<th>Acciones</th>` condicionado por `{{if and $.Session (ge $.Session.Nivel 2)}}`.
- Cada `<tr>` recibe `id="tarima-{{.ID}}"` para que htmx pueda apuntar al target correcto.
- Botón de eliminar (solo nivel >= 2):
  ```html
  <button type="button" class="btn btn-danger btn-sm"
      onclick="confirmarEliminar({{.ID}})"
      title="Eliminar tarima">
      <i class="fas fa-trash"></i>
  </button>
  ```
- Cards mobile: mismo botón trash condicionado por nivel.
- Actualizar `colspan` del empty state de 10 → 11.

### 6.2 `web/templates/tarimas.html`
- Modal Bootstrap de confirmación (fuera del contenido, al final del template):
  - Header rojo con título "Confirmar Eliminación".
  - Body: "¿Estás seguro de eliminar esta tarima? Se registrará en el historial de eliminaciones."
  - Footer: botones "Cancelar" (cierra modal) y "Eliminar" (dispara `htmx.ajax('DELETE', ...)`).

## 7. JavaScript `web/static/shared.js`

- `confirmarEliminar(id)`: abre el modal Bootstrap, almacena el ID en variable global.
- En `DOMContentLoaded`: listener en el botón "Confirmar" que cierra el modal y ejecuta `htmx.ajax('DELETE', '/tarimas/' + id, {target: '#tarima-' + id, swap: 'delete'})`.
- Listener `hidden.bs.modal` para limpiar la variable global.

## 8. Orden de ejecución

| # | Archivo | Acción |
|---|---|---|
| 1 | `docs/PLAN-F4.md` | Crear este plan |
| 2 | `internal/tarima/repository.go` | Agregar `GetByIDRaw` + `Delete` a la interfaz |
| 3 | `internal/tarima/service.go` | Implementar método `Delete` |
| 4 | `internal/tarima/service_test.go` | Tests de Delete con mock |
| 5 | `internal/infrastructure/sqlite/tarima_repository.go` | Implementar `GetByIDRaw` + `Delete` con transacción |
| 6 | `internal/infrastructure/sqlite/tarima_repository_test.go` | Tests de Delete + verificación de historial |
| 7 | `internal/delivery/handlers/tarima.go` | Handler `EliminarTarima` |
| 8 | `main.go` | Registrar ruta `DELETE /tarimas/{id}` nivel 2 |
| 9 | `web/templates/partials/tabla_tarimas.html` | Columna acciones + `id` en `<tr>` + botón trash |
| 10 | `web/templates/tarimas.html` | Modal Bootstrap de confirmación |
| 11 | `web/static/shared.js` | Función `confirmarEliminar` + listeners |
| 12 | Verificación | `go vet`, `go build`, `go test`, smoke test |

## 9. Verificación F4
```
go vet ./...
go build ./...
go test ./...
go run .
# Smoke tests:
GET /tarimas                   → 302 a /login (sin sesión)
POST /login (admin/password)   → HX-Redirect /
GET /tarimas                   → 200 con columna Acciones (admin nivel 4)
DELETE /tarimas/3              → 200, tarima eliminada de la tabla
DELETE /tarimas/99999          → 404
# Verificar en SQLite:
SELECT * FROM historial_tarimas → registro con todos los campos copiados
SELECT * FROM tarimas          → tarima eliminada ya no aparece
```

## 10. Límites de F4
- Sin visualización de historial (pendiente).
- Sin paginación (mismo comportamiento que PHP).
- Confirmación con modal Bootstrap (no modal nativo del browser).
- Sin soft delete: la eliminación es física (la tarima se borra de la tabla y se copia al historial).

## 11. Detalles técnicos resueltos durante la implementación
- **`GetByIDRaw` vs `GetByID`**: se usa la tabla `tarimas` directamente (no la vista) porque la vista hace JOIN con `usuarios` y podría fallar si el usuario fue eliminado. La vista se mantiene para listados y filtrado.
- **Transacción en el repo (no en el servicio)**: la transacción se encapsula en `Delete` del repo porque el servicio solo coordina el flujo. El repo tiene acceso directo a `*sql.DB` y puede `BeginTx`.
- **200 vs 204 para htmx**: `204 No Content` desactiva el swap en htmx; `200 OK` con body vacío + `hx-swap="delete"` elimina el `<tr>` del DOM correctamente.
- **`defer tx.Rollback()`**: en SQLite con WAL y `BEGIN` normal, si `Commit` ya se ejecutó, `Rollback` es un no-op. Si falla antes, hace rollback automático.
- **`RowsAffected` después de DELETE**: verifica que exactamente 1 fila fue borrada (si es 0, la tarima no existía; si es > 1, algo está mal con la integridad).
