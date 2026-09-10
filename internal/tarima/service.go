package tarima

import "context"

const DefaultLimit = 1000
const MaxLimit = 10000

type TarimaService struct {
	repo TarimaRepository
}

func NewTarimaService(repo TarimaRepository) *TarimaService {
	return &TarimaService{repo: repo}
}

func (s *TarimaService) List(ctx context.Context, filters FiltrosTarima, showAll bool, limit int) ([]Tarima, error) {
	if limit <= 0 {
		limit = DefaultLimit
	} else if limit > MaxLimit {
		limit = MaxLimit
	}

	if filters.HasFilters() {
		return s.repo.ListFiltered(ctx, filters, limit)
	}
	if showAll {
		return s.repo.ListAll(ctx, limit)
	}
	return s.repo.ListToday(ctx, limit)
}

func (s *TarimaService) CountToday(ctx context.Context) (int, error) {
	return s.repo.CountToday(ctx)
}
