package producto

import "context"

type ProductoRepository interface {
	ListAll(ctx context.Context) ([]Producto, error)
}
