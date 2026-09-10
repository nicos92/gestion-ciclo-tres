package tarima

import "context"

type TarimaRepository interface {
	ListToday(ctx context.Context, limit int) ([]Tarima, error)
	ListAll(ctx context.Context, limit int) ([]Tarima, error)
	ListFiltered(ctx context.Context, filters FiltrosTarima, limit int) ([]Tarima, error)
	CountToday(ctx context.Context) (int, error)
}
