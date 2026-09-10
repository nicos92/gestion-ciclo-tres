# Plan detallado — Fase 0 (F0): Esqueleto + DB + base Docker

## Objetivo
Levantar la aplicación Go "Gestión de Tarimas" (módulo `gestion-ciclo-tres`) con configuración por entorno, base de datos SQLite migrada y seedada, endpoint `GET /healthz` funcional y apagado graceful. Sin lógica de negocio (login, tarimas, etc. entran en F1+).

## Criterio de aceptación (def-de-done)
- `go vet ./...` sin errores.
- `go build ./...` y `CGO_ENABLED=0 go build ./...` (binario estático → base Docker lista).
- `go test ./...` OK: config + migración/seed.
- `go run .` levanta; `GET /healthz` responde `200` con ping a DB; la DB contiene el seed (4 roles, 2 usuarios, 2 tarimas).

## 1. Limpieza previa
- Borrar el demo todo: `main.go`, `model.go`, `repository.go`, `service.go`, `handlers.go`, `render.go` y `templates/`.
- `go.mod`: renombrar a `module gestion-ciclo-tres` y agregar dependencias:
  - `modernc.org/sqlite` (puro Go, `CGO_ENABLED=0` → igual en Windows, Linux y contenedor).
  - `golang.org/x/crypto` (bcrypt; se usa en el seed de F0 y en login en F1).
- `.gitignore`: agregar `/data/` (la DB local `./data/gestiontarimas.db` no se commitea).

## 2. Rol IDs y niveles
| id | nombre_rol | nivel |
|---|---|---|
| 1 | administrador | 4 |
| 2 | jefe_produccion | 3 |
| 3 | supervisor | 2 |
| 4 | produccion | 1 (default en `usuarios`) |

RBAC por rango: crear/ver = nivel 1, eliminar = 2, editar = 3, gestionar usuarios = 4.

## 3. Archivos nuevos y responsabilidad

| Archivo | Responsabilidad |
|---|---|
| `main.go` | Composition root: embed de `migrations/*.sql`, `config.Load()`, `time.Local` (TZ), abrir+migrar+seed la DB, registrar handlers, `http.Server` con timeouts, graceful shutdown con `signal.NotifyContext` (SIGINT/SIGTERM), logging con `log/slog` |
| `internal/config/config.go` | Struct `Config{Port, DBPath, TZ, AppName}`; `Load() (Config, error)` leyendo env con defaults; normaliza `PORT` (`8080`→`:8080`); valida rango de puerto y TZ (`time.LoadLocation`) |
| `internal/config/config_test.go` | Defaults, override por env, PORT inválido, TZ inválido |
| `internal/infrastructure/sqlite/sqlite.go` | `Open(path string) (*sql.DB, error)`: driver `sqlite` (modernc), DSN con pragmas, `SetMaxOpenConns`/`SetMaxIdleConns` |
| `internal/infrastructure/sqlite/migrate.go` | `Migrate(ctx, db, fsys fs.FS) error`: crea `schema_migrations`, aplica `*.sql` ordenadas, cada una en transacción + registro de versión; idempotente |
| `internal/infrastructure/sqlite/seed.go` | `Seed(ctx, db) error`: si roles vacíos → error/aviso; si usuarios vacíos → inserta `admin` y `produccion` con bcrypt Go, más 2 tarimas de ejemplo |
| `internal/infrastructure/sqlite/sqlite_test.go` | DB en `t.TempDir()`: migración aplicada (tablas + 4 roles) |
| `internal/infrastructure/sqlite/seed_test.go` | Usuarios creados y password verificado con `bcrypt.CompareHashAndPassword`; idempotencia (correr 2 veces) |
| `internal/delivery/handlers/healthz.go` | `GET /healthz`: `db.PingContext` → `200 "ok"` / `503` |
| `migrations/0001_init.sql` | Schema (roles, usuarios, tarimas, historial_tarimas, índices) |
| `migrations/0002_seed.sql` | Datos estáticos sin secretos: los 4 roles |

El embed de `migrations/*.sql` va en `main.go` (raíz) y `Migrate` recibe un `fs.FS` → testeable con su propio FS.

## 4. Conexión SQLite (decisión de robustez)
- Driver: `modernc.org/sqlite` (`_ "modernc.org/sqlite"`).
- DSN: `file:<ruta normalizada con forward slashes>?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate`.
- Concurrencia: WAL + `busy_timeout` + `_txlock=immediate` permiten pool >1. Si en fases posteriores aparecen `SQLITE_BUSY`, fallback simple `db.SetMaxOpenConns(1)`. Decisión documentada.
- `path` → normalizar `\` a `/` (Windows) para que el mismo DSN funcione en Linux.

## 5. Migraciones
- `Migrate` crea `schema_migrations(version TEXT PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)` fuera de transacción.
- Por cada archivo `NNNN_*.sql` (orden lexicográfico): si no está en `schema_migrations` → BEGIN · exec contenido · INSERT versión · COMMIT.
- Multi-statement por `Exec`: modernc lo soporta; si algún driver lo rechazara, split por `;` (se verifica en F0).

### `0001_init.sql` (resumen)
```sql
roles(id INTEGER PK AUTOINCREMENT, nombre_rol TEXT UNIQUE NOT NULL, nivel INTEGER NOT NULL,
      descripcion TEXT, activo INTEGER DEFAULT 1);

usuarios(id INTEGER PK, username TEXT UNIQUE NOT NULL, email TEXT UNIQUE NOT NULL,
         password TEXT NOT NULL, first_name TEXT NOT NULL, last_name TEXT NOT NULL,
         legajo TEXT NOT NULL, department TEXT, id_rol INTEGER DEFAULT 4, activo INTEGER DEFAULT 1,
         created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
         FOREIGN KEY(id_rol) REFERENCES roles(id));

tarimas(id INTEGER PK, codigo_barras TEXT UNIQUE NOT NULL, numero_producto TEXT NOT NULL,
        numero_tarima TEXT NOT NULL, numero_usuario TEXT NOT NULL, cantidad_cajas INTEGER NOT NULL,
        peso NUMERIC DEFAULT 0, numero_venta TEXT NOT NULL, descripcion TEXT, id_usuario INTEGER,
        fecha_registro DATETIME DEFAULT CURRENT_TIMESTAMP, fecha DATE DEFAULT (date('now')),
        FOREIGN KEY(id_usuario) REFERENCES usuarios(id) ON DELETE SET NULL);

historial_tarimas(id INTEGER PK, id_tarima_eliminada INTEGER NOT NULL, codigo_barras TEXT,
                  numero_producto TEXT, numero_tarima TEXT, numero_usuario TEXT, cantidad_cajas INTEGER,
                  peso NUMERIC, numero_venta TEXT, descripcion TEXT, id_usuario INTEGER,
                  fecha_registro DATETIME, fecha DATE, fecha_eliminacion DATETIME DEFAULT CURRENT_TIMESTAMP);

CREATE INDEX idx_tarimas_codigo_barras ON tarimas(codigo_barras);
CREATE INDEX idx_tarimas_numero_tarima ON tarimas(numero_tarima);
CREATE INDEX idx_tarimas_fecha_registro ON tarimas(fecha_registro);
CREATE INDEX idx_tarimas_id_usuario ON tarimas(id_usuario);
```
Notas: sin tablas `permisos`/`roles_permisos` (RBAC por nivel); `legajo` como TEXT (el dump real es varchar(20), no INT); `updated_at` se setea manualmente en los UPDATE (F1+); `fecha DATE DEFAULT (date('now'))`.

### `0002_seed.sql` — roles
```sql
INSERT INTO roles (id, nombre_rol, nivel, descripcion) VALUES
(1, 'administrador',   4, 'Usuario con permisos totales'),
(2, 'jefe_produccion', 3, 'Jefe de producción con permisos para gestionar tarimas'),
(3, 'supervisor',      2, 'Supervisor con permisos para eliminar tarimas'),
(4, 'produccion',      1, 'Usuario con permisos limitados para operaciones de producción');
```

### Seed programático (`seed.go`) — bcrypt Go
- Si `COUNT(roles) < 4` → error (el seed SQL no corrió o quedó incompleto).
- Si `COUNT(usuarios) = 0`:
  - `admin` (email `admin@empresa.com`, password `password`, bcrypt) rol administrador (1), activo.
  - `produccion` (email `produccion@empresa.com`, password bcrypt) rol produccion (4), activo.
- Si `COUNT(tarimas) = 0`: 2 tarimas de ejemplo referenciando a `admin` (id 1), con códigos de barras de 30 dígitos válidos del dump PHP.
- Idempotente; ninguna operación PHP para hashes (solo bcrypt de Go).

## 6. main.go (solo infraestructura)
- `//go:embed migrations/*.sql` → `fsys embed.FS`.
- `config.Load()` → `time.Local = LoadLocation(cfg.TZ)`.
- `os.MkdirAll(dirname)` de `DB_PATH` si corresponde.
- `sqlite.Open(cfg.DBPath)` → `sqlite.Migrate(db, fsys)` → `sqlite.Seed(db)` → `db.Close()` diferido.
- `mux`: `/healthz` → `handlers.NewHealthz(db)`.
- `http.Server{Addr: cfg.Port, ReadHeaderTimeout, ReadTimeout, WriteTimeout, IdleTimeout}`.
- Goroutine `ListenAndServe` + `signal.NotifyContext(SIGINT, SIGTERM)`; al señal: `srv.Shutdown(ctx 10s)` y cierre de DB. Logs con `slog` (arranque, addr, db path, shutdown, errores).

## 7. Verificación F0
```
go vet ./...
go build ./...
CGO_ENABLED=0 go build ./...
go test ./...
go run .
# luego:
Invoke-WebRequest http://localhost:8080/healthz   → 200
```

## 8. Límites de F0
- Sin handlers de negocio, templates ni sesión (F1+).
- Sin lógica de tarimas/usuarios (F2+).
- Sin Dockerfile/compose (F6); F0 solo garantiza binario estático (CGO=0) y `/data/` ignorado.

## 9. Riesgos a vigilar
- DSN en Windows (barra invertida → forward slashes): cubierto por el test en `t.TempDir()`.
- Multi-statement por `Exec` con modernc: verificado en `migrate_test`.
- `SQLITE_BUSY` futuro bajo htmx: mitigation WAL/busy_timeout/txlock; fallback pool=1.
- TZ afecta el filtro "hoy" de F2; configurable por env `TZ` (default `America/Argentina/Buenos_Aires`).

## 10. Dependencias a agregar
- `modernc.org/sqlite`
- `golang.org/x/crypto` (bcrypt, seed + login F1)

## Orden de ejecución F0
1. Limpieza demo + `go.mod` (`gestion-ciclo-tres`) + deps.
2. `internal/config` + tests.
3. `internal/infrastructure/sqlite` (sqlite.go, migrate.go, seed.go) + tests.
4. `internal/delivery/handlers/healthz.go`.
5. `migrations/` (0001_init.sql, 0002_seed.sql).
6. `main.go`.
7. `.gitignore` (+ `/data/`).
8. Verificación completa.