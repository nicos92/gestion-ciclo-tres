package tarima

import (
	"context"
	"testing"
)

type mockRepo struct {
	todayCalled    bool
	allCalled      bool
	filteredCalled bool
	limit          int
	filters        FiltrosTarima
}

func (m *mockRepo) ListToday(_ context.Context, limit int) ([]Tarima, error) {
	m.todayCalled = true
	m.limit = limit
	return []Tarima{{ID: 1}}, nil
}

func (m *mockRepo) ListAll(_ context.Context, limit int) ([]Tarima, error) {
	m.allCalled = true
	m.limit = limit
	return []Tarima{{ID: 1}, {ID: 2}}, nil
}

func (m *mockRepo) ListFiltered(_ context.Context, filters FiltrosTarima, limit int) ([]Tarima, error) {
	m.filteredCalled = true
	m.filters = filters
	m.limit = limit
	return []Tarima{{ID: 3}}, nil
}

func (m *mockRepo) CountToday(_ context.Context) (int, error) {
	return 5, nil
}

func TestList_DefaultToday(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	result, err := svc.List(context.Background(), FiltrosTarima{}, false, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !repo.todayCalled {
		t.Error("expected today query")
	}
	if len(result) != 1 {
		t.Errorf("got %d results, want 1", len(result))
	}
	if repo.limit != DefaultLimit {
		t.Errorf("limit = %d, want %d", repo.limit, DefaultLimit)
	}
}

func TestList_ShowAll(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	result, err := svc.List(context.Background(), FiltrosTarima{}, true, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !repo.allCalled {
		t.Error("expected all query")
	}
	if len(result) != 2 {
		t.Errorf("got %d results, want 2", len(result))
	}
}

func TestList_FiltersActive(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	filters := FiltrosTarima{NumeroProducto: "123"}
	result, err := svc.List(context.Background(), filters, true, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !repo.filteredCalled {
		t.Error("expected filtered query")
	}
	if repo.filters.NumeroProducto != "123" {
		t.Errorf("filters.NumeroProducto = %q, want 123", repo.filters.NumeroProducto)
	}
	if len(result) != 1 {
		t.Errorf("got %d results, want 1", len(result))
	}
}

func TestList_FiltersTakesPrecedenceOverShowAll(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	filters := FiltrosTarima{Legajo: "A001"}
	result, err := svc.List(context.Background(), filters, true, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !repo.filteredCalled {
		t.Error("expected filtered query (filters takes precedence)")
	}
	if repo.allCalled {
		t.Error("should not call all when filters are active")
	}
	if repo.limit != 100 {
		t.Errorf("limit = %d, want 100", repo.limit)
	}
	_ = result
}

func TestList_CapsLimit(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	svc.List(context.Background(), FiltrosTarima{}, false, 99999)
	if repo.limit != MaxLimit {
		t.Errorf("limit = %d, want MaxLimit %d", repo.limit, MaxLimit)
	}

	smallRepo := &mockRepo{}
	smallSvc := NewTarimaService(smallRepo)
	smallSvc.List(context.Background(), FiltrosTarima{}, false, -1)
	if smallRepo.limit != DefaultLimit {
		t.Errorf("limit = %d, want DefaultLimit %d", smallRepo.limit, DefaultLimit)
	}
}

func TestCountToday(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	n, err := svc.CountToday(context.Background())
	if err != nil {
		t.Fatalf("CountToday: %v", err)
	}
	if n != 5 {
		t.Errorf("CountToday = %d, want 5", n)
	}
}

func TestHasFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters FiltrosTarima
		want    bool
	}{
		{"empty", FiltrosTarima{}, false},
		{"producto", FiltrosTarima{NumeroProducto: "123"}, true},
		{"tarima", FiltrosTarima{NumeroTarima: "456"}, true},
		{"usuario", FiltrosTarima{NumeroUsuario: "01"}, true},
		{"venta", FiltrosTarima{NumeroVenta: "25-123"}, true},
		{"fecha", FiltrosTarima{FechaRegistro: "2025-01-01"}, true},
		{"legajo", FiltrosTarima{Legajo: "A001"}, true},
		{"nombre", FiltrosTarima{NombreUsuario: "Juan"}, true},
		{"cajas_min", func() FiltrosTarima { v := 10; return FiltrosTarima{CantidadCajasMin: &v} }(), true},
		{"peso_min", func() FiltrosTarima { v := 100.5; return FiltrosTarima{PesoMin: &v} }(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filters.HasFilters(); got != tt.want {
				t.Errorf("HasFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}
