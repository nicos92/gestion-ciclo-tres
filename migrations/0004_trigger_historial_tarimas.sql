CREATE TRIGGER IF NOT EXISTS trg_historial_tarimas
AFTER DELETE ON tarimas
BEGIN
    INSERT INTO historial_tarimas (
        id_tarima_eliminada, codigo_barras, numero_producto, numero_tarima,
        numero_usuario, conservacion, cantidad_cajas, peso, numero_venta,
        descripcion, id_usuario, fecha_registro, fecha
    ) VALUES (
        OLD.id, OLD.codigo_barras, OLD.numero_producto, OLD.numero_tarima,
        OLD.numero_usuario, OLD.conservacion, OLD.cantidad_cajas, OLD.peso,
        OLD.numero_venta, OLD.descripcion, OLD.id_usuario, OLD.fecha_registro,
        OLD.fecha
    );
END;