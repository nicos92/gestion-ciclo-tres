package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"gestion-ciclo-tres/internal/producto"
)

type SQLiteProductoRepository struct {
	db *sql.DB
}

func NewProductoRepository(db *sql.DB) *SQLiteProductoRepository {
	return &SQLiteProductoRepository{db: db}
}

func (r *SQLiteProductoRepository) ListAll(ctx context.Context) ([]producto.Producto, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT numero_producto, nombre
		FROM productos
		ORDER BY CAST(numero_producto AS INTEGER)`)
	if err != nil {
		return nil, fmt.Errorf("consultar catálogo de productos: %w", err)
	}
	defer rows.Close()

	var productos []producto.Producto
	for rows.Next() {
		var p producto.Producto
		if err := rows.Scan(&p.Numero, &p.Nombre); err != nil {
			return nil, fmt.Errorf("leer producto del catálogo: %w", err)
		}
		productos = append(productos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer catálogo de productos: %w", err)
	}
	return productos, nil
}
