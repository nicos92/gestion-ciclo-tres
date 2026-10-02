package sqlite

import (
	"context"
	"strconv"
	"testing"
)

func newTestProductoRepo(t *testing.T) (context.Context, *SQLiteProductoRepository) {
	t.Helper()
	ctx, tdb := newTestDB(t)
	return ctx, NewProductoRepository(tdb.db)
}

func TestProductoListAll_CatalogoSembrado(t *testing.T) {
	ctx, repo := newTestProductoRepo(t)

	productos, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(productos) != 670 {
		t.Fatalf("ListAll devolvió %d productos, want 670", len(productos))
	}
	if productos[0].Numero != "2" || productos[0].Nombre != "SERVICIO EN SEGURIDAD E HIGIENE" {
		t.Errorf("primer producto = %+v, want el número 2", productos[0])
	}
}

func TestProductoListAll_OrdenNumerico(t *testing.T) {
	ctx, repo := newTestProductoRepo(t)

	productos, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	// El orden lexicográfico de la columna TEXT pondría "1000" antes que "126".
	prev := -1
	for _, p := range productos {
		n, err := strconv.Atoi(p.Numero)
		if err != nil {
			t.Fatalf("número de producto %q no es numérico: %v", p.Numero, err)
		}
		if n <= prev {
			t.Fatalf("orden ascendente roto: %d aparece después de %d", n, prev)
		}
		prev = n
	}
}

func TestProductoListAll_AltaVisible(t *testing.T) {
	ctx, tdb := newTestProductoRepo(t)
	repo := NewProductoRepository(tdb.db)

	if _, err := tdb.db.ExecContext(ctx,
		`INSERT INTO productos (numero_producto, nombre) VALUES (?, ?)`, "42", "PRODUCTO NUEVO"); err != nil {
		t.Fatalf("insertar producto: %v", err)
	}

	productos, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(productos) != 671 {
		t.Fatalf("ListAll devolvió %d productos, want 671", len(productos))
	}

	var encontrado bool
	for _, p := range productos {
		if p.Numero == "42" {
			encontrado = true
			if p.Nombre != "PRODUCTO NUEVO" {
				t.Errorf("nombre = %q, want PRODUCTO NUEVO", p.Nombre)
			}
		}
	}
	if !encontrado {
		t.Error("el producto dado de alta no apareció en ListAll")
	}
}
