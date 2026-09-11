package tarima

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

type mockRepo struct {
	todayCalled    bool
	allCalled      bool
	filteredCalled bool
	limit          int
	filters        FiltrosTarima
	createCalled   bool
	getByIDCalled  bool
	updateCalled   bool
	deleteCalled   bool
	createID       int64
	tarimas        []Tarima
	singleTarima   *Tarima
	createErr      error
	getErr         error
	updateErr      error
	deleteErr      error
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

func (m *mockRepo) Create(_ context.Context, t *Tarima) (int64, error) {
	m.createCalled = true
	if m.createErr != nil {
		return 0, m.createErr
	}
	return m.createID, nil
}

func (m *mockRepo) GetByID(_ context.Context, id int64) (*Tarima, error) {
	m.getByIDCalled = true
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.singleTarima != nil {
		return m.singleTarima, nil
	}
	return &Tarima{ID: id, CodigoBarras: "08801970009998010100450450000025-123456"}, nil
}

func (m *mockRepo) Update(_ context.Context, t *Tarima) error {
	m.updateCalled = true
	return m.updateErr
}

func (m *mockRepo) GetByIDRaw(_ context.Context, id int64) (*Tarima, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.singleTarima != nil {
		return m.singleTarima, nil
	}
	return &Tarima{ID: id, CodigoBarras: "08801970009998010100450450000025-123456"}, nil
}

func (m *mockRepo) Delete(_ context.Context, id int64) (*Tarima, error) {
	m.deleteCalled = true
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	return &Tarima{ID: id, CodigoBarras: "08801970009998010100450450000025-123456"}, nil
}

func (m *mockRepo) CountAll(_ context.Context) (int, error) {
	return 10, nil
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

func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{createID: 10}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}

	id, err := svc.Create(context.Background(), tarima)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !repo.createCalled {
		t.Error("expected repo.Create to be called")
	}
	if id != 10 {
		t.Errorf("id = %d, want 10", id)
	}
}

func TestCreate_CamposRequeridos(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroVenta:    "25-123456",
	}
	_, err := svc.Create(context.Background(), tarima)
	if err != ErrCodigoBarrasRequerido {
		t.Errorf("got %v, want ErrCodigoBarrasRequerido", err)
	}
}

func TestCreate_Duplicado(t *testing.T) {
	repo := &mockRepo{createErr: fmt.Errorf("UNIQUE constraint failed: tarimas.codigo_barras")}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	_, err := svc.Create(context.Background(), tarima)
	if err != ErrCodigoBarrasDuplicado {
		t.Errorf("got %v, want ErrCodigoBarrasDuplicado", err)
	}
}

func TestCreate_BarcodeNoCoincide(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "111111",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	_, err := svc.Create(context.Background(), tarima)
	if err != ErrBarcodeNoCoincide {
		t.Errorf("got %v, want ErrBarcodeNoCoincide", err)
	}
	if repo.createCalled {
		t.Error("repo.Create should not be called when barcode does not match fields")
	}
}

func TestCreate_AutoRellenoDesdeBarcode(t *testing.T) {
	repo := &mockRepo{createID: 11}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		CodigoBarras: "088019700099999981010045045000",
		NumeroVenta:  "25-123456",
	}
	id, err := svc.Create(context.Background(), tarima)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != 11 {
		t.Errorf("id = %d, want 11", id)
	}
	if tarima.NumeroProducto != "880197" ||
		tarima.NumeroTarima != "000999" ||
		tarima.NumeroUsuario != "010" ||
		tarima.Conservacion != "1" ||
		tarima.CantidadCajas != 45 ||
		tarima.Peso != 450.00 {
		t.Errorf("campos no auto-rellenados correctamente: %+v", tarima)
	}
}

func TestGetByID_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	tarima, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !repo.getByIDCalled {
		t.Error("expected repo.GetByID to be called")
	}
	if tarima == nil {
		t.Fatal("expected non-nil tarima")
	}
	if tarima.ID != 1 {
		t.Errorf("ID = %d, want 1", tarima.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{getErr: sql.ErrNoRows}
	svc := NewTarimaService(repo)

	_, err := svc.GetByID(context.Background(), 999)
	if err != ErrTarimaNoEncontrada {
		t.Errorf("got %v, want ErrTarimaNoEncontrada", err)
	}
}

func TestUpdate_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		ID:             1,
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	err := svc.Update(context.Background(), tarima)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !repo.updateCalled {
		t.Error("expected repo.Update to be called")
	}
}

func TestUpdate_Duplicado(t *testing.T) {
	repo := &mockRepo{updateErr: fmt.Errorf("UNIQUE constraint failed: tarimas.codigo_barras")}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		ID:             1,
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	err := svc.Update(context.Background(), tarima)
	if err != ErrCodigoBarrasDuplicado {
		t.Errorf("got %v, want ErrCodigoBarrasDuplicado", err)
	}
}

func TestUpdate_BarcodeNoCoincide(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	tarima := &Tarima{
		ID:             1,
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		CantidadCajas:  45,
		Peso:           999.99,
		NumeroVenta:    "25-123456",
	}
	err := svc.Update(context.Background(), tarima)
	if err != ErrBarcodeNoCoincide {
		t.Errorf("got %v, want ErrBarcodeNoCoincide", err)
	}
	if repo.updateCalled {
		t.Error("repo.Update should not be called when barcode does not match fields")
	}
}

func TestDelete_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTarimaService(repo)

	result, err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !repo.deleteCalled {
		t.Error("expected repo.Delete to be called")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != 1 {
		t.Errorf("ID = %d, want 1", result.ID)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockRepo{getErr: sql.ErrNoRows}
	svc := NewTarimaService(repo)

	_, err := svc.Delete(context.Background(), 999)
	if err != ErrTarimaNoEncontrada {
		t.Errorf("got %v, want ErrTarimaNoEncontrada", err)
	}
	if repo.deleteCalled {
		t.Error("repo.Delete should not be called when GetByIDRaw fails")
	}
}
