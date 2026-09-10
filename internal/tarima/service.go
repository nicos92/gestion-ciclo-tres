package tarima

import (
	"context"
	"database/sql"
	"strings"
)

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

func (s *TarimaService) Create(ctx context.Context, t *Tarima) (int64, error) {
	trimTarima(t)

	if len(t.CodigoBarras) == 30 {
		if bd, err := ParseBarcode(t.CodigoBarras); err == nil {
			if t.NumeroProducto == "" {
				t.NumeroProducto = bd.NumeroProducto
			}
			if t.NumeroTarima == "" {
				t.NumeroTarima = bd.NumeroTarima
			}
			if t.NumeroUsuario == "" {
				t.NumeroUsuario = bd.NumeroUsuario
			}
			if t.Conservacion == "" {
				t.Conservacion = bd.Conservacion
			}
			if t.CantidadCajas == 0 {
				t.CantidadCajas = bd.CantidadCajas
			}
			if t.Peso == 0 {
				t.Peso = bd.Peso
			}
		}
	}

	if err := ValidateTarima(t); err != nil {
		return 0, err
	}

	id, err := s.repo.Create(ctx, t)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, ErrCodigoBarrasDuplicado
		}
		return 0, err
	}
	return id, nil
}

func (s *TarimaService) GetByID(ctx context.Context, id int64) (*Tarima, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTarimaNoEncontrada
		}
		return nil, err
	}
	return t, nil
}

func (s *TarimaService) Update(ctx context.Context, t *Tarima) error {
	trimTarima(t)

	if err := ValidateTarima(t); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, t); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrCodigoBarrasDuplicado
		}
		return err
	}
	return nil
}

func trimTarima(t *Tarima) {
	t.CodigoBarras = strings.TrimSpace(t.CodigoBarras)
	t.NumeroProducto = strings.TrimSpace(t.NumeroProducto)
	t.NumeroTarima = strings.TrimSpace(t.NumeroTarima)
	t.NumeroUsuario = strings.TrimSpace(t.NumeroUsuario)
	t.Conservacion = strings.TrimSpace(t.Conservacion)
	t.NumeroVenta = strings.TrimSpace(t.NumeroVenta)
	t.Descripcion = strings.TrimSpace(t.Descripcion)
}
