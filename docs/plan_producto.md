
# 1. Análisis de Impacto y Requerimientos

* **Modelo Relacional:** Se debe crear una nueva tabla `productos` que actuará como catálogo maestro. La tabla existente `tarima` pasará a tener una clave foránea que relacione su columna `numero_producto` con la clave primaria de la tabla `productos`.
* **Restricciones de SQLite:** SQLite tiene soporte limitado para `ALTER TABLE` al modificar restricciones de claves foráneas en tablas existentes. Por lo tanto, se debe seguir el procedimiento seguro de migración (crear nueva tabla temporal, copiar datos, renombrar).
* **Seed de Datos:** El proceso de inicialización (`seed`) de la base de datos debe ampliarse para poblar primero la tabla `productos` con el listado inicial antes de insertar o relacionar las tarimas.

---

## 2. Planificación de Pasos

1. **Diseño del Esquema:**

* Definir la tabla `productos` con `numero_producto` (Primary Key, tipo adecuado como INTEGER o TEXT) y `nombre` (TEXT, NOT NULL).
* Actualizar la tabla `tarima` para definir `numero_producto` como Foreign Key apuntando a `productos(numero_producto)`.

1. **Estrategia de Migración / Seed en Go:**

* Verificar si la tabla `productos` ya existe o limpiar/inicializar la base de datos de desarrollo.
* Modificar la función de `seed` para insertar los registros iniciales de productos.
* Asegurar que las inserciones en `tarima` respeten la nueva relación (o actualizar el seed de tarimas si fuera necesario).

1. **Adaptación en la Aplicación (Go + HTMX):**

* Revisar los modelos (`structs` en Go) y las consultas SQL (repositorios) para incluir la relación con productos si las vistas HTMX lo requieren (por ejemplo, mostrar el nombre del producto en las tarimas).

---

## 3. Guía de Desarrollo para el Agente

### Paso 1: Definición de Tablas (SQL)

Implementa la creación de la tabla y la reestructuración de `tarima` asegurando que las claves foráneas estén activas (`PRAGMA foreign_keys = ON;`):

```sql
-- Crear la tabla productos
CREATE TABLE IF NOT EXISTS productos (
    numero_producto INTEGER PRIMARY KEY,
    nombre TEXT NOT NULL
);

-- Si la tabla tarima ya existe y no tiene la FK, aplicar migración segura:
CREATE TABLE tarima_nueva (
    -- definir aquí las columnas actuales de tarima, asegurando:
    numero_producto INTEGER,
    FOREIGN KEY (numero_producto) REFERENCES productos(numero_producto)
    -- ... resto de columnas
);
-- Copiar datos, eliminar vieja y renombrar nueva (si aplica)

```

### Paso 2: Actualización del Seed en Go

Modifica la rutina de inicialización de la base de datos para insertar los productos antes de procesar las tarimas:

```go
func SeedProductos(db *sql.DB) error {
    query := `INSERT OR IGNORE INTO productos (numero_producto, nombre) VALUES (?, ?)`
    
    productos := []struct {
        Numero int
        Nombre string
    }{
        {1, "Producto A"},
        {2, "Producto B"},
        -- Agregar el listado completo requerido
    }

    for _, p := range productos {
        _, err := db.Exec(query, p.Numero, p.Nombre)
        if err != nil {
            return err
        }
    }
    return nil
}

```

### Paso 3: Validación y Ajustes en Endpoints / HTMX

* Asegúrate de que las consultas que listan las tarimas realicen un `JOIN` con la tabla `productos` si la interfaz HTMX necesita renderizar el nombre del producto en lugar de solo su número.
