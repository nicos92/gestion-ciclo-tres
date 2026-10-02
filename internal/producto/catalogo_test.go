package producto

import (
	"context"
	"errors"
	"testing"
)

type mockRepo struct {
	productos []Producto
	err       error
	calls     int
}

func (m *mockRepo) ListAll(_ context.Context) ([]Producto, error) {
	m.calls++
	return m.productos, m.err
}

func nuevoCatalogoDePrueba(t *testing.T) (*Catalogo, *mockRepo) {
	t.Helper()
	repo := &mockRepo{productos: []Producto{
		{Numero: "2", Nombre: "SERVICIO EN SEGURIDAD E HIGIENE"},
		{Numero: "20", Nombre: "MEDIA RES 8C S/DEL"},
		{Numero: "100", Nombre: "PISTOLA 3/4C"},
	}}
	c, err := NewCatalogo(context.Background(), repo)
	if err != nil {
		t.Fatalf("NewCatalogo: %v", err)
	}
	return c, repo
}

func TestNewCatalogo_CargaAlConstruir(t *testing.T) {
	c, repo := nuevoCatalogoDePrueba(t)

	if repo.calls != 1 {
		t.Errorf("ListAll llamado %d veces, want 1", repo.calls)
	}
	if len(c.Todos()) != 3 {
		t.Errorf("Todos() devolvió %d productos, want 3", len(c.Todos()))
	}
}

func TestNewCatalogo_ErrorDelRepositorio(t *testing.T) {
	repo := &mockRepo{err: errors.New("boom")}

	if _, err := NewCatalogo(context.Background(), repo); err == nil {
		t.Error("NewCatalogo no devolvió error")
	}
}

func TestBuscar_ResuelveNumeroExacto(t *testing.T) {
	c, _ := nuevoCatalogoDePrueba(t)

	p, ok := c.Buscar("100")
	if !ok {
		t.Fatal("Buscar(100) no encontró el producto")
	}
	if p.Nombre != "PISTOLA 3/4C" {
		t.Errorf("nombre = %q, want PISTOLA 3/4C", p.Nombre)
	}
}

// El código de barras siempre entrega el producto con padding de ceros
// (raw[1:7]), así que "000002" tiene que resolver el producto "2".
func TestBuscar_IgnoraCerosIniciales(t *testing.T) {
	c, _ := nuevoCatalogoDePrueba(t)

	p, ok := c.Buscar("000002")
	if !ok {
		t.Fatal("Buscar(000002) no encontró el producto 2")
	}
	if p.Nombre != "SERVICIO EN SEGURIDAD E HIGIENE" {
		t.Errorf("nombre = %q, want SERVICIO EN SEGURIDAD E HIGIENE", p.Nombre)
	}
}

func TestBuscar_IgnoraEspacios(t *testing.T) {
	c, _ := nuevoCatalogoDePrueba(t)

	if _, ok := c.Buscar(" 20 "); !ok {
		t.Error("Buscar(\" 20 \") no encontró el producto 20")
	}
}

func TestBuscar_NoEncontrado(t *testing.T) {
	c, _ := nuevoCatalogoDePrueba(t)

	for _, numero := range []string{"999999", "0", "abc", ""} {
		if p, ok := c.Buscar(numero); ok {
			t.Errorf("Buscar(%q) encontró %+v, want not found", numero, p)
		}
	}
}

func TestReload_PublicaAltaNueva(t *testing.T) {
	c, repo := nuevoCatalogoDePrueba(t)

	repo.productos = append(repo.productos, Producto{Numero: "999999", Nombre: "MULTI-PRODUCTO"})
	if err := c.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if len(c.Todos()) != 4 {
		t.Errorf("Todos() devolvió %d productos, want 4", len(c.Todos()))
	}
	p, ok := c.Buscar("999999")
	if !ok {
		t.Fatal("Buscar(999999) no encontró el producto dado de alta")
	}
	if p.Nombre != "MULTI-PRODUCTO" {
		t.Errorf("nombre = %q, want MULTI-PRODUCTO", p.Nombre)
	}
}

func TestReload_ErrorConservaSnapshotAnterior(t *testing.T) {
	c, repo := nuevoCatalogoDePrueba(t)

	repo.err = errors.New("boom")
	if err := c.Reload(context.Background()); err == nil {
		t.Fatal("Reload no devolvió error")
	}

	if len(c.Todos()) != 3 {
		t.Errorf("Tras un Reload fallido quedaron %d productos, want 3", len(c.Todos()))
	}
	if _, ok := c.Buscar("100"); !ok {
		t.Error("Tras un Reload fallido el snapshot anterior se perdió")
	}
}

func TestCatalogoVacio(t *testing.T) {
	repo := &mockRepo{}
	c, err := NewCatalogo(context.Background(), repo)
	if err != nil {
		t.Fatalf("NewCatalogo: %v", err)
	}

	if len(c.Todos()) != 0 {
		t.Errorf("Todos() devolvió %d productos, want 0", len(c.Todos()))
	}
	if _, ok := c.Buscar("2"); ok {
		t.Error("Buscar encontró un producto en un catálogo vacío")
	}
}
