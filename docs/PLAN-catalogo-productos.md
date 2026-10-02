# Plan detallado — Catálogo de productos en memoria + nombre de producto en el formulario

## Objetivo

Mantener el catálogo de la tabla `productos` cargado en memoria desde el arranque del servidor, y usarlo para que, al completar el código de barras en las pantallas de nueva y editar tarima, el nombre del producto se resuelva en el servidor y se muestre en un campo de texto de sólo lectura.

Hoy el nombre del producto se resuelve en SQL, dentro del `JOIN` de las vistas, y el formulario no lo muestra en absoluto.

## Análisis del estado actual

### La tabla ya existe, pero ningún código Go la lee

`migrations/0006_productos.sql:1-5` define el catálogo maestro:

```sql
CREATE TABLE IF NOT EXISTS productos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    numero_producto TEXT UNIQUE NOT NULL,
    nombre          TEXT NOT NULL
);
```

con 670 registros sembrados en la misma migración (`sqlite_test.go:126` lo afirma).

`productos` **nunca se consulta desde Go**. El nombre del producto llega a la aplicación únicamente porque las vistas hacen el join:

- `migrations/0007_vista_tarimas_con_producto.sql:20-23` → `vista_tarimas_con_legajo`
- `migrations/0007_vista_tarimas_con_producto.sql:46-49` → `vista_historial_tarimas`

En ambos casos con la misma expresión:

```sql
COALESCE(p.nombre, '') AS nombre_producto
...
LEFT JOIN productos p ON CAST(t.numero_producto AS INTEGER) = CAST(p.numero_producto AS INTEGER)
```

Ese `nombre_producto` lo expone `tarimaSelectCols` (`sqlite/tarima_repository.go:21-24`) y el scanner lo carga en `Tarima.NombreProducto` (`internal/tarima/tarima.go:45`) y `TarimaEliminada.NombreProducto` (`tarima.go:66`). Los templates lo muestran en las tablas y tarjetas (`partials/tabla_tarimas.html:50,115`).

### El formulario no consulta nada

`web/templates/partials/form_tarimas.html:33-35` es un input numérico pelado, sin catálogo:

```html
<input type="number" class="form-control" id="numeroProducto" name="numeroProducto"
       placeholder="Producto" required step="1" maxlength="6"
       value="{{if .Tarima}}{{.Tarima.NumeroProducto}}{{end}}">
```

El mismo partial alimenta las pantallas de nueva (`nueva_tarima.html:36`) y editar (`editar_tarima.html:38`), así que un único cambio cubre ambas.

### Hallazgo crítico: la clave del catálogo es numérica

El número de producto sale del código de barras como `raw[1:7]` (`internal/tarima/barcode.go:40`): **6 dígitos con padding de ceros**. El catálogo guarda `"2"`, `"20"`, `"999999"`.

Por eso las vistas comparan con `CAST(... AS INTEGER)`: `CAST('000002' AS INTEGER)` = `CAST('2' AS INTEGER)`.

**Consecuencia:** el índice en memoria tiene que usar clave numérica normalizada. Con clave de string crudo, `000002` no encontraría `2` y el campo quedaría vacío ante un barcode perfectamente válido.

### Puntos de inserción

| Necesidad | Ubicación |
|---|---|
| Cargar el catálogo al arranque | `main.go:104-171`, después de `sqlite.Seed` (líneas 125-127) |
| Pasar datos al template del form | `internal/delivery/handlers/tarima.go:43-51` (`tarimaFormData`) |
| Disparar la consulta al completar el barcode | `web/static/shared.js:193` (`autoFillFromBarcode`) |
| Registrar la ruta | `main.go:238-311` (bloque de rutas de tarimas) |

`run()` es el composition root tanto en modo proceso como en modo Windows service (`service_windows.go:46`), así que inicializar ahí cubre ambos.

## Decisiones de diseño

### Snapshot inmutable con swap atómico

El servidor atiende cada request en su propia goroutine, así que el catálogo se lee concurrentemente. Se optó por un snapshot inmutable publicado con `atomic.Pointer`:

```go
type Catalogo struct {
    repo     ProductoRepository
    snapshot atomic.Pointer[Snapshot]
}

type Snapshot struct {
    productos []Producto
    porNumero map[string]Producto
}
```

- `Buscar` / `Todos` hacen un único `Load()` por llamada: **sin locks**, seguro para lectura concurrente. Mismo criterio que `render.Renderer`, que se construye una vez y después sólo se lee.
- `Reload(ctx)` reconstruye el snapshot completo y lo intercambia.

El soporte para recarga se incluye desde ahora porque está previsto dar de alta productos más adelante (`Catalogo` conserva la referencia al repositorio justamente para eso). Hoy `main.go` sólo llama a `NewCatalogo` una vez; cuando exista el alta de productos, alcanza con agregar `Reload` sin tocar el resto del código.

La normalización de clave (`000002` → `2`) vive dentro de `Catalogo`, de modo que el handler HTTP no necesita conocer ese detalle.

### Consultar por evento `change`, no `keyup`

El número de producto no se tipea: se completa desde el código de barras. Se usa `hx-trigger="change"` para no generar un request por tecla, y el evento se dispara desde JS al cerrar el barcode.

En JS, **setear `.value` no dispara `change`**, así que htmx no se enteraría sin un dispatch explícito. Se agrega en `shared.js:193`:

```js
producto.dispatchEvent(new Event('change'));
```

Con eso el barcode es el disparador y el tipeo manual del producto funciona por el mismo camino, sin duplicar lógica.

### El campo de nombre no se persiste

La tabla `tarimas` no tiene columna de nombre de producto: el nombre se deriva por `JOIN`. El input de sólo lectura va **sin atributo `name`**, por lo que no viaja en el POST y no corre riesgo de persistirse.

## Criterio de aceptación (def-done)

- `go vet ./...`, `go build ./...`, `go test ./...` sin errores.
- El catálogo se carga una vez al arranque; si la carga falla, la aplicación no levanta.
- `GET /tarimas/nueva`: al escribir un código de barras válido de 30 dígitos, el campo de nombre se completa con el nombre del catálogo.
- `GET /tarimas/editar/{id}`: el campo de nombre viene precargado con `Tarima.NombreProducto`.
- Con un número de producto que no existe en el catálogo, el campo queda vacío mostrando el placeholder.
- El campo de nombre no es editable ni focusable.
- Ningún dato nuevo se escribe en la base.

## 1. Dominio `internal/producto/` (nuevo)

### 1.1 `producto.go`
```go
type Producto struct {
    Numero string
    Nombre string
}
```

### 1.2 `repository.go`
```go
type ProductoRepository interface {
    ListAll(ctx context.Context) ([]Producto, error)
}
```

### 1.3 `catalogo.go`
- `Catalogo`, `Snapshot`, `NewCatalogo`, `Reload`, `Todos`, `Buscar`.
- Índice por clave numérica normalizada.
- `Todos` devuelve el slice del snapshot (sólo lectura); documentarlo así para no inducir mutaciones.
- Normalización: `strconv.Atoi` → `strconv.Itoa`, con fallback al string recortado si no es numérico.

### 1.4 `catalogo_test.go`
Carga desde un repositorio fake, `Buscar("000002")` resuelve `"2"`, número inexistente, catálogo vacío, y `Reload` posterior a un alta.

## 2. Infraestructura `internal/infrastructure/sqlite/`

### 2.1 `producto_repository.go` (nuevo)
```sql
SELECT numero_producto, nombre FROM productos
ORDER BY CAST(numero_producto AS INTEGER)
```
El orden es numérico para que salga `2, 20, 100, 126` y no el lexicográfico `100, 1000, 126, 20`.

### 2.2 `producto_repository_test.go` (nuevo)
Integración contra SQLite temporal, siguiendo el patrón de `tarima_repository_test.go`.

## 3. Delivery `internal/delivery/handlers/`

### 3.1 `producto.go` (nuevo)
- `ProductoHandler` con `renderer` + `catalogo`.
- `Buscar`: lee el número del producto, resuelve contra el catálogo y devuelve el fragmento `campo_nombre_producto`. Responde `200` en ambos casos (con o sin nombre) para que el swap sea siempre válido.
- Struct `campoProductoData{Numero, Nombre}`, compartido por este handler y por `TarimaHandler` para no duplicar el tipo.

### 3.2 `tarima.go` (modificado)
- `tarimaFormData` suma `CampoProducto *campoProductoData`.
- Helper `newFormData(session, editMode)` que reemplaza la repetición de los 7 campos en los 6 puntos de construcción actuales (líneas 96, 163, 210, 223, 311, 323) y arma el campo de producto.

## 4. Rutas `main.go`
- Construir `productoRepo` + `catalogo` después de `Seed`; loguear la cantidad cargada; error de carga = fatal.
- `GET /productos/buscar` con `requireNivel(NivelProduccion)`, junto al bloque de tarimas.
- La URL no lleva el número en el path: htmx lo envía como query param con el `name` del input (`numeroProducto`), así la consulta es siempre dinámica.

## 5. Templates
- `partials/campo_nombre_producto.html` (nuevo): `{{define "campo_nombre_producto"}}` con el input `readonly`, `tabindex="-1"`, `bg-body-secondary` (Bootstrap 5.3.0) y placeholder "Sin nombre asignado en el catálogo".
- `partials/form_tarimas.html` (modificado): `hx-get` / `hx-trigger` / `hx-target` / `hx-swap` en `#numeroProducto`, más `{{template "campo_nombre_producto" .CampoProducto}}` debajo.

El partial se registra solo y se concatena en el parse set de cada página (`render/render.go:83-153`), sin cambios en el renderer.

`hx-swap="outerHTML"` sobre el contenedor `#campo-nombre-producto`: reemplaza el `input-group` completo y mantiene el layout de la grilla.

## 6. JavaScript `web/static/shared.js`
- Dispatch de `change` sobre `#numeroProducto` al final del autofill del código de barras.

## 7. Verificación
`go vet ./...` · `go build ./...` · `go test ./...`

Smoke test manual: cargar `/tarimas/nueva`, escribir un barcode de 30 dígitos con `9998` en las posiciones 13-17, confirmar que el nombre se completa; repetir con un número de producto inexistente para ver el placeholder; repetir sobre `/tarimas/editar/{id}` para validar la precarga.

## Fuera de alcance

- **No se toca el `JOIN` de las vistas.** Siguen siendo la fuente del nombre para los listados, y quedan coherentes con el catálogo porque ambos leen de la misma tabla.
- **No se valida en el backend** que el número de producto exista en el catálogo. El campo es `required` pero el número es libre; se puede guardar un producto inexistente. Es una decisión explícita.
- **No se agregan pantallas de administración de productos.** El soporte para dar de alta productos queda preparado (`Catalogo.Reload`) pero no implementado.
- Inconsistencia preexistente que no se toca: `CountToday` cuenta sobre la vista mientras el listado trae hasta 1000 filas (`tarima.DefaultLimit`), así que "Tarimas mostradas" y "Tarimas ingresadas hoy" pueden diferir.