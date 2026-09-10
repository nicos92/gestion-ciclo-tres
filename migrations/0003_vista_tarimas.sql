CREATE VIEW IF NOT EXISTS vista_tarimas_con_legajo AS
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
    COALESCE(u.first_name || ' ' || u.last_name, '') AS nombre_usuario
FROM tarimas t
LEFT JOIN usuarios u ON t.id_usuario = u.id;
