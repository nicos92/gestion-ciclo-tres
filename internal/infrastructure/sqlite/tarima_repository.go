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
	cantidad_cajas, peso, numero_venta, descripcion, id_usuario,
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
			&t.CantidadCajas, &t.Peso, &t.NumeroVenta, &t.Descripcion, &idUsuario,
			&fechaRegistroStr, &fechaStr, &t.Legajo, &t.NombreUsuario,
		); err != nil {
			return nil, fmt.Errorf("leer fila tarima: %w", err)
		}
		if idUsuario.Valid {
			t.IDUsuario = &idUsuario.Int64
		}
		// Los timestamps almacenados son UTC (CURRENT_TIMESTAMP); se parsean
		// como UTC para que formatDate los convierta a time.Local (TZ).
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

// parseSQLTimestamp interpreta la marca devuelta por el driver (RFC3339 UTC)
// como UTC para que formatDate la convierta a time.Local (TZ). Soporta
// también "YYYY-MM-DD HH:MM:SS" como fallback. Si falla, devuelve zero time.
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

// parseSQLDate interpreta "YYYY-MM-DD"(o RFC3339) como UTC. Si falla, zero time.
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