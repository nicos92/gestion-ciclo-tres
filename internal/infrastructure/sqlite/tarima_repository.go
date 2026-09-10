package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gestion-ciclo-tres/internal/tarima"
)

type SQLiteTarimaRepository struct {
	db *sql.DB
}

func NewTarimaRepository(db *sql.DB) *SQLiteTarimaRepository {
	return &SQLiteTarimaRepository{db: db}
}

const tarimaSelectCols = `
	id, codigo_barras, numero_producto, numero_tarima, numero_usuario,
	conservacion, cantidad_cajas, peso, numero_venta, descripcion, id_usuario,
	fecha_registro, fecha, legajo, nombre_usuario`

func (r *SQLiteTarimaRepository) ListToday(ctx context.Context, limit int) ([]tarima.Tarima, error) {
	query := `
		SELECT ` + tarimaSelectCols + `
		FROM vista_tarimas_con_legajo
		WHERE date(fecha_registro, 'localtime') = date('now', 'localtime')
		ORDER BY fecha_registro DESC
		LIMIT ?`
	return r.queryTarimas(ctx, query, limit)
}

func (r *SQLiteTarimaRepository) ListAll(ctx context.Context, limit int) ([]tarima.Tarima, error) {
	query := `
		SELECT ` + tarimaSelectCols + `
		FROM vista_tarimas_con_legajo
		ORDER BY fecha_registro DESC
		LIMIT ?`
	return r.queryTarimas(ctx, query, limit)
}

func (r *SQLiteTarimaRepository) ListFiltered(ctx context.Context, filters tarima.FiltrosTarima, limit int) ([]tarima.Tarima, error) {
	query, args := buildFilterQuery(filters, limit)
	return r.queryTarimasArgs(ctx, query, args)
}

func (r *SQLiteTarimaRepository) CountToday(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM vista_tarimas_con_legajo WHERE date(fecha_registro, 'localtime') = date('now', 'localtime')`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("contar tarimas de hoy: %w", err)
	}
	return n, nil
}

func (r *SQLiteTarimaRepository) CountAll(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tarimas`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("contar tarimas: %w", err)
	}
	return n, nil
}

func (r *SQLiteTarimaRepository) Create(ctx context.Context, t *tarima.Tarima) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tarimas (codigo_barras, numero_producto, numero_tarima, numero_usuario,
		                     conservacion, cantidad_cajas, peso, numero_venta, descripcion, id_usuario)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.CodigoBarras, t.NumeroProducto, t.NumeroTarima, t.NumeroUsuario,
		t.Conservacion, t.CantidadCajas, t.Peso, t.NumeroVenta, t.Descripcion, t.IDUsuario)
	if err != nil {
		return 0, fmt.Errorf("insertar tarima: %w", err)
	}
	return res.LastInsertId()
}

func (r *SQLiteTarimaRepository) GetByID(ctx context.Context, id int64) (*tarima.Tarima, error) {
	query := `SELECT ` + tarimaSelectCols + ` FROM vista_tarimas_con_legajo WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var t tarima.Tarima
	var idUsuario sql.NullInt64
	var fechaRegistroStr, fechaStr string
	err := row.Scan(
		&t.ID, &t.CodigoBarras, &t.NumeroProducto, &t.NumeroTarima, &t.NumeroUsuario,
		&t.Conservacion, &t.CantidadCajas, &t.Peso, &t.NumeroVenta, &t.Descripcion, &idUsuario,
		&fechaRegistroStr, &fechaStr, &t.Legajo, &t.NombreUsuario,
	)
	if err != nil {
		return nil, err
	}
	if idUsuario.Valid {
		t.IDUsuario = &idUsuario.Int64
	}
	t.FechaRegistro = parseSQLTimestamp(fechaRegistroStr)
	t.Fecha = parseSQLDate(fechaStr)
	return &t, nil
}

func (r *SQLiteTarimaRepository) Update(ctx context.Context, t *tarima.Tarima) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tarimas
		SET codigo_barras = ?, numero_producto = ?, numero_tarima = ?, numero_usuario = ?,
		    conservacion = ?, cantidad_cajas = ?, peso = ?, numero_venta = ?, descripcion = ?
		WHERE id = ?`,
		t.CodigoBarras, t.NumeroProducto, t.NumeroTarima, t.NumeroUsuario,
		t.Conservacion, t.CantidadCajas, t.Peso, t.NumeroVenta, t.Descripcion,
		t.ID)
	if err != nil {
		return fmt.Errorf("actualizar tarima: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar actualización: %w", err)
	}
	if n == 0 {
		return tarima.ErrTarimaNoEncontrada
	}
	return nil
}

const tarimaRawCols = `
	id, codigo_barras, numero_producto, numero_tarima, numero_usuario,
	conservacion, cantidad_cajas, peso, numero_venta, descripcion, id_usuario,
	fecha_registro, fecha`

func (r *SQLiteTarimaRepository) GetByIDRaw(ctx context.Context, id int64) (*tarima.Tarima, error) {
	query := `SELECT ` + tarimaRawCols + ` FROM tarimas WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var t tarima.Tarima
	var idUsuario sql.NullInt64
	var fechaRegistroStr, fechaStr string
	err := row.Scan(
		&t.ID, &t.CodigoBarras, &t.NumeroProducto, &t.NumeroTarima, &t.NumeroUsuario,
		&t.Conservacion, &t.CantidadCajas, &t.Peso, &t.NumeroVenta, &t.Descripcion, &idUsuario,
		&fechaRegistroStr, &fechaStr,
	)
	if err != nil {
		return nil, err
	}
	if idUsuario.Valid {
		t.IDUsuario = &idUsuario.Int64
	}
	t.FechaRegistro = parseSQLTimestamp(fechaRegistroStr)
	t.Fecha = parseSQLDate(fechaStr)
	return &t, nil
}

func (r *SQLiteTarimaRepository) Delete(ctx context.Context, id int64) (*tarima.Tarima, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	var t tarima.Tarima
	var idUsuario sql.NullInt64
	var fechaRegistroStr, fechaStr string

	err = tx.QueryRowContext(ctx, `SELECT `+tarimaRawCols+` FROM tarimas WHERE id = ?`, id).Scan(
		&t.ID, &t.CodigoBarras, &t.NumeroProducto, &t.NumeroTarima, &t.NumeroUsuario,
		&t.Conservacion, &t.CantidadCajas, &t.Peso, &t.NumeroVenta, &t.Descripcion, &idUsuario,
		&fechaRegistroStr, &fechaStr,
	)
	if err != nil {
		return nil, err
	}
	if idUsuario.Valid {
		t.IDUsuario = &idUsuario.Int64
	}
	t.FechaRegistro = parseSQLTimestamp(fechaRegistroStr)
	t.Fecha = parseSQLDate(fechaStr)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO historial_tarimas (
			id_tarima_eliminada, codigo_barras, numero_producto, numero_tarima,
			numero_usuario, conservacion, cantidad_cajas, peso, numero_venta,
			descripcion, id_usuario, fecha_registro, fecha
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.CodigoBarras, t.NumeroProducto, t.NumeroTarima,
		t.NumeroUsuario, t.Conservacion, t.CantidadCajas, t.Peso, t.NumeroVenta,
		t.Descripcion, t.IDUsuario, t.FechaRegistro, t.Fecha)
	if err != nil {
		return nil, fmt.Errorf("insertar en historial: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM tarimas WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("eliminar tarima: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("verificar eliminación: %w", err)
	}
	if n == 0 {
		return nil, tarima.ErrTarimaNoEncontrada
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &t, nil
}

func (r *SQLiteTarimaRepository) queryTarimas(ctx context.Context, query string, limit int) ([]tarima.Tarima, error) {
	return r.queryTarimasArgs(ctx, query, []interface{}{limit})
}

func (r *SQLiteTarimaRepository) queryTarimasArgs(ctx context.Context, query string, args []interface{}) ([]tarima.Tarima, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("consultar tarimas: %w", err)
	}
	defer rows.Close()

	var result []tarima.Tarima
	for rows.Next() {
		var t tarima.Tarima
		var idUsuario sql.NullInt64
		var fechaRegistroStr, fechaStr string
		if err := rows.Scan(
			&t.ID, &t.CodigoBarras, &t.NumeroProducto, &t.NumeroTarima, &t.NumeroUsuario,
			&t.Conservacion, &t.CantidadCajas, &t.Peso, &t.NumeroVenta, &t.Descripcion, &idUsuario,
			&fechaRegistroStr, &fechaStr, &t.Legajo, &t.NombreUsuario,
		); err != nil {
			return nil, fmt.Errorf("leer fila tarima: %w", err)
		}
		if idUsuario.Valid {
			t.IDUsuario = &idUsuario.Int64
		}
		t.FechaRegistro = parseSQLTimestamp(fechaRegistroStr)
		t.Fecha = parseSQLDate(fechaStr)
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar tarimas: %w", err)
	}
	return result, nil
}

func buildFilterQuery(filters tarima.FiltrosTarima, limit int) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}

	sb.WriteString(`SELECT ` + tarimaSelectCols + ` FROM vista_tarimas_con_legajo WHERE 1=1`)

	if filters.NumeroProducto != "" {
		sb.WriteString(` AND numero_producto LIKE '%' || ? || '%'`)
		args = append(args, filters.NumeroProducto)
	}
	if filters.NumeroTarima != "" {
		sb.WriteString(` AND numero_tarima LIKE '%' || ? || '%'`)
		args = append(args, filters.NumeroTarima)
	}
	if filters.NumeroUsuario != "" {
		sb.WriteString(` AND numero_usuario LIKE '%' || ? || '%'`)
		args = append(args, filters.NumeroUsuario)
	}
	if filters.NumeroVenta != "" {
		sb.WriteString(` AND numero_venta LIKE '%' || ? || '%'`)
		args = append(args, filters.NumeroVenta)
	}
	if filters.FechaRegistro != "" {
		sb.WriteString(` AND date(fecha_registro, 'localtime') = ?`)
		args = append(args, filters.FechaRegistro)
	}
	if filters.Legajo != "" {
		sb.WriteString(` AND legajo LIKE '%' || ? || '%'`)
		args = append(args, filters.Legajo)
	}
	if filters.NombreUsuario != "" {
		sb.WriteString(` AND nombre_usuario LIKE '%' || ? || '%'`)
		args = append(args, filters.NombreUsuario)
	}
	if filters.CantidadCajasMin != nil {
		sb.WriteString(` AND cantidad_cajas >= ?`)
		args = append(args, *filters.CantidadCajasMin)
	}
	if filters.PesoMin != nil {
		sb.WriteString(` AND peso >= ?`)
		args = append(args, *filters.PesoMin)
	}

	sb.WriteString(` ORDER BY fecha_registro DESC LIMIT ?`)
	args = append(args, limit)

	return sb.String(), args
}

func parseSQLTimestamp(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseSQLDate(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}
