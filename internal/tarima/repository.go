package tarima

import "context"

type TarimaRepository interface {
	ListToday(ctx context.Context, limit int) ([]Tarima, error)
	ListAll(ctx context.Context, limit int) ([]Tarima, error)
	ListFiltered(ctx context.Context, filters FiltrosTarima, limit int) ([]Tarima, error)
	CountToday(ctx context.Context) (int, error)
	CountAll(ctx context.Context) (int, error)
	Create(ctx context.Context, t *Tarima) (int64, error)
	GetByID(ctx context.Context, id int64) (*Tarima, error)
	GetByIDRaw(ctx context.Context, id int64) (*Tarima, error)
	Update(ctx context.Context, t *Tarima) error
	Delete(ctx context.Context, id int64) (*Tarima, error)
	ListHistorial(ctx context.Context, filters FiltrosHistorial, limit int) ([]TarimaEliminada, error)
}
