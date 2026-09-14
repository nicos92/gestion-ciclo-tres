CREATE VIEW IF NOT EXISTS vista_historial_tarimas AS
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
    COALESCE(u.first_name || ' ' || u.last_name, '') AS nombre_usuario
FROM historial_tarimas h
LEFT JOIN usuarios u ON h.id_usuario = u.id;
