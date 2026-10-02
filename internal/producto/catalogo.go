package producto

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
)

// snapshot es una copia inmutable del catálogo en un momento dado. Se publica
// completa mediante un swap, de modo que las lecturas concurrentes no toman lock.
type snapshot struct {
	productos []Producto
	porNumero map[string]Producto
}

// Catalogo mantiene el catálogo de productos en memoria para resolver nombres
// sin consultar la base en cada request. Es seguro para uso concurrente.
type Catalogo struct {
	repo    ProductoRepository
	current atomic.Pointer[snapshot]
}

func NewCatalogo(ctx context.Context, repo ProductoRepository) (*Catalogo, error) {
	c := &Catalogo{repo: repo}
	if err := c.Reload(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// Reload relee el catálogo y publica el snapshot nuevo. Se usa al dar de alta
// productos; mientras no se llame, se sigue sirviendo el snapshot cargado al
// arranque.
func (c *Catalogo) Reload(ctx context.Context) error {
	productos, err := c.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("cargar catálogo de productos: %w", err)
	}

	snap := &snapshot{
		productos: productos,
		porNumero: make(map[string]Producto, len(productos)),
	}
	for _, p := range productos {
		snap.porNumero[normalize(p.Numero)] = p
	}
	c.current.Store(snap)
	return nil
}

// Todos devuelve el catálogo completo en orden numérico. El slice es de solo
// lectura y se comparte entre requests.
func (c *Catalogo) Todos() []Producto {
	return c.current.Load().productos
}

// Buscar resuelve un número de producto. El número puede venir con ceros a la
// izquierda (el código de barras siempre los trae) y se normaliza igual que el
// CAST de las vistas SQL.
func (c *Catalogo) Buscar(numero string) (Producto, bool) {
	p, ok := c.current.Load().porNumero[normalize(numero)]
	return p, ok
}

func normalize(numero string) string {
	trimmed := strings.TrimSpace(numero)
	if n, err := strconv.Atoi(trimmed); err == nil {
		return strconv.Itoa(n)
	}
	return trimmed
}
