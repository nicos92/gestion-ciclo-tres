DROP VIEW IF EXISTS vista_tarimas_con_legajo;

CREATE VIEW vista_tarimas_con_legajo AS
SELECT
    t.id,
    t.codigo_barras,
    t.numero_producto,
    t.numero_tarima,
    t.numero_usuario,
    COALESCE(t.conservacion, '') AS conservacion,
    t.cantidad_cajas,
    t.peso,
    t.numero_venta,
    COALESCE(t.descripcion, '') AS descripcion,
    t.id_usuario,
    t.fecha_registro,
    t.fecha,
    COALESCE(u.legajo, '') AS legajo,
    COALESCE(u.first_name || ' ' || u.last_name, '') AS nombre_usuario,
    COALESCE(p.nombre, '') AS nombre_producto
FROM tarimas t
LEFT JOIN usuarios u ON t.id_usuario = u.id
LEFT JOIN productos p ON CAST(t.numero_producto AS INTEGER) = CAST(p.numero_producto AS INTEGER);

DROP VIEW IF EXISTS vista_historial_tarimas;

CREATE VIEW vista_historial_tarimas AS
SELECT
    h.id,
    h.id_tarima_eliminada,
    h.codigo_barras,
    h.numero_producto,
    h.numero_tarima,
    h.numero_usuario,
    COALESCE(h.conservacion, '') AS conservacion,
    h.cantidad_cajas,
    h.peso,
    h.numero_venta,
    COALESCE(h.descripcion, '') AS descripcion,
    h.id_usuario,
    h.fecha_registro,
    h.fecha,
    h.fecha_eliminacion,
    COALESCE(u.legajo, '') AS legajo,
    COALESCE(u.first_name || ' ' || u.last_name, '') AS nombre_usuario,
    COALESCE(p.nombre, '') AS nombre_producto
FROM historial_tarimas h
LEFT JOIN usuarios u ON h.id_usuario = u.id
LEFT JOIN productos p ON CAST(h.numero_producto AS INTEGER) = CAST(p.numero_producto AS INTEGER);