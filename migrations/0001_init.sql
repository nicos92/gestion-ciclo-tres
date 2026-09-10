CREATE TABLE roles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre_rol  TEXT UNIQUE NOT NULL,
    nivel       INTEGER NOT NULL,
    descripcion TEXT,
    activo      INTEGER DEFAULT 1
);

CREATE TABLE usuarios (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT UNIQUE NOT NULL,
    email       TEXT UNIQUE NOT NULL,
    password    TEXT NOT NULL,
    first_name  TEXT NOT NULL,
    last_name   TEXT NOT NULL,
    legajo      TEXT NOT NULL,
    department  TEXT,
    id_rol      INTEGER DEFAULT 4,
    activo      INTEGER DEFAULT 1,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_rol) REFERENCES roles (id)
);

CREATE TABLE tarimas (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    codigo_barras   TEXT UNIQUE NOT NULL,
    numero_producto TEXT NOT NULL,
    numero_tarima   TEXT NOT NULL,
    numero_usuario  TEXT NOT NULL,
    conservacion    TEXT DEFAULT '',
    cantidad_cajas  INTEGER NOT NULL,
    peso            NUMERIC DEFAULT 0,
    numero_venta    TEXT NOT NULL,
    descripcion     TEXT,
    id_usuario      INTEGER,
    fecha_registro  DATETIME DEFAULT CURRENT_TIMESTAMP,
    fecha           DATE DEFAULT (date('now')),
    FOREIGN KEY (id_usuario) REFERENCES usuarios (id) ON DELETE SET NULL
);

CREATE TABLE historial_tarimas (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    id_tarima_eliminada INTEGER NOT NULL,
    codigo_barras     TEXT,
    numero_producto   TEXT,
    numero_tarima     TEXT,
    numero_usuario    TEXT,
    conservacion      TEXT,
    cantidad_cajas    INTEGER,
    peso              NUMERIC,
    numero_venta      TEXT,
    descripcion       TEXT,
    id_usuario        INTEGER,
    fecha_registro    DATETIME,
    fecha             DATE,
    fecha_eliminacion DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tarimas_codigo_barras ON tarimas (codigo_barras);
CREATE INDEX idx_tarimas_numero_tarima ON tarimas (numero_tarima);
CREATE INDEX idx_tarimas_fecha_registro ON tarimas (fecha_registro);
CREATE INDEX idx_tarimas_id_usuario ON tarimas (id_usuario);
CREATE INDEX idx_historial_fecha_eliminacion ON historial_tarimas (fecha_eliminacion);