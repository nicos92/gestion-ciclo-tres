package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"gestion-ciclo-tres/internal/tarima"
)

func newTestTarimaRepo(t *testing.T) (context.Context, *SQLiteTarimaRepository) {
	t.Helper()
	ctx, tdb := newTestDB(t)
	return ctx, NewTarimaRepository(tdb.db)
}

// buildBarcode arma un código de barras de 30 dígitos (formato de la app):
// 1 dígito "0" + 6 de producto + 6 de tarima + "9998" + 1 de conservación + 3 de usuario + 3 de cajas + 6 de peso.
func buildBarcode(numeroProducto, numeroTarima, numeroUsuario, conservacion string, cajas int, peso float64) (string, error) {
	pesoCentavos := int(peso*100 + 0.5)
	barcode := "0" +
		numeroProducto + numeroTarima +
		"9998" + conservacion + numeroUsuario +
		fmt.Sprintf("%03d", cajas) +
		fmt.Sprintf("%06d", pesoCentavos)
	if len(barcode) != 30 {
		return "", fmt.Errorf("barcode de %d dígitos, se esperaban 30", len(barcode))
	}
	return barcode, nil
}

func (r *SQLiteTarimaRepository) insertTestTarima(ctx context.Context, t *testing.T, numeroProducto, numeroTarima, numeroUsuario, conservacion string, cajas int, peso float64, numeroVenta string, idUsuario *int64, fechaRegistro string) {
	t.Helper()
	codigo, err := buildBarcode(numeroProducto, numeroTarima, numeroUsuario, conservacion, cajas, peso)
	if err != nil {
		t.Fatalf("buildBarcode: %v", err)
	}
	venta := numeroVenta
	if venta == "" {
		venta = "25-000000"
	}

	stmt := `INSERT INTO tarimas (codigo_barras, numero_producto, numero_tarima, numero_usuario,
	               conservacion, cantidad_cajas, peso, numero_venta, id_usuario, fecha_registro)
	       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, stmt,
		codigo, numeroProducto, numeroTarima, numeroUsuario, conservacion, cajas, peso, venta, idUsuario, fechaRegistro); err != nil {
		t.Fatalf("insertar tarima de test: %v", err)
	}
}

func TestListToday_OnlyToday(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	ayer := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "100000", "100001", "050", "1", 10, 100.5, "25-000001", nil, ayer)

	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "100000", "100002", "050", "1", 11, 101.5, "25-000002", nil, hoy)

	tarimas, err := repo.ListToday(ctx, 1000)
	if err != nil {
		t.Fatalf("ListToday: %v", err)
	}
	if len(tarimas) != 1 {
		t.Errorf("ListToday = %d tarimas, se esperaba 1", len(tarimas))
	}
}

func TestListAll_ReturnsAll(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	ayer := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	adminID := int64(1)
	repo.insertTestTarima(ctx, t, "200000", "200001", "060", "2", 5, 50.0, "25-000002", &adminID, ayer)

	tarimas, err := repo.ListAll(ctx, 1000)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(tarimas) != 1 {
		t.Errorf("ListAll = %d tarimas, se esperaban 1", len(tarimas))
	}
}

func TestListFiltered_ByProducto(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)
	adminID := int64(1)
	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "777888", "300001", "080", "1", 12, 80.0, "25-000003", &adminID, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{NumeroProducto: "777888"}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by producto: %v", err)
	}
	if len(tarimas) != 1 {
		t.Fatalf("ListFiltered = %d, se esperaba 1", len(tarimas))
	}
	if tarimas[0].NumeroTarima != "300001" {
		t.Errorf("NumeroTarima = %q, want 300001", tarimas[0].NumeroTarima)
	}
}

func TestListFiltered_ByLegajoConJoin(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)
	adminID := int64(1)

	ayer := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "111222", "400001", "090", "1", 20, 200.0, "25-000004", &adminID, ayer)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{Legajo: "A0001"}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by legajo: %v", err)
	}
	if len(tarimas) < 1 {
		t.Fatalf("ListFiltered by legajo = 0, se esperaba >= 1")
	}
	for _, tt := range tarimas {
		if tt.Legajo != "A0001" {
			t.Errorf("Legajo = %q, want A0001", tt.Legajo)
		}
		if tt.NombreUsuario == "" {
			t.Error("NombreUsuario vacío, se esperaba el nombre del admin")
		}
	}
}

func TestListFiltered_ByCajasMin(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)
	minCajas := 40

	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "333444", "500001", "010", "1", 50, 10.0, "25-000005", nil, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{CantidadCajasMin: &minCajas}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by cajas min: %v", err)
	}
	if len(tarimas) != 1 {
		t.Errorf("ListFiltered cajas>=40 = %d, se esperaba 1", len(tarimas))
	}
	if tarimas[0].CantidadCajas != 50 {
		t.Errorf("CantidadCajas = %d, want 50", tarimas[0].CantidadCajas)
	}
}

func TestListFiltered_ByPesoMin(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)
	minPeso := 400.0

	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "555666", "600001", "020", "1", 60, 600.0, "25-000006", nil, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{PesoMin: &minPeso}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by peso min: %v", err)
	}
	// Seed eliminado: solo la insertada con 600.0.
	if len(tarimas) != 1 {
		t.Errorf("ListFiltered peso>=400 = %d, se esperaba 1", len(tarimas))
	}
}

func TestListFiltered_ByFecha(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	ayer := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	ayerUTC := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")

	repo.insertTestTarima(ctx, t, "777111", "700001", "030", "1", 99, 999.0, "25-000007", nil, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{FechaRegistro: ayer}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by fecha: %v", err)
	}
	if len(tarimas) != 0 {
		t.Errorf("fecha %s = %d tarimas, se esperaba 0", ayer, len(tarimas))
	}

	repo.insertTestTarima(ctx, t, "777222", "700002", "031", "2", 5, 5.0, "25-000008", nil, ayerUTC)

	tarimas, err = repo.ListFiltered(ctx, tarima.FiltrosTarima{FechaRegistro: ayer}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by fecha (2da consulta): %v", err)
	}
	if len(tarimas) != 1 {
		t.Errorf("fecha %s = %d tarimas, se esperaba 1", ayer, len(tarimas))
	}
}

func TestCountToday(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "900000", "900001", "070", "1", 12, 120.0, "25-000010", nil, hoy)

	n, err := repo.CountToday(ctx)
	if err != nil {
		t.Fatalf("CountToday: %v", err)
	}
	if n != 1 {
		t.Errorf("CountToday = %d, se esperaba 1", n)
	}
}

func TestListLimit(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	for i := 0; i < 3; i++ {
		repo.insertTestTarima(ctx, t, "111000", fmt.Sprintf("8000%02d", i), "011", "1", 1, 1.0, "25-000009", nil,
			time.Now().UTC().Format("2006-01-02 15:04:05"))
	}

	tarimas, err := repo.ListToday(ctx, 3)
	if err != nil {
		t.Fatalf("ListToday with limit: %v", err)
	}
	if len(tarimas) != 3 {
		t.Errorf("ListToday limit 3 = %d, se esperaban 3", len(tarimas))
	}
}

func TestCreate_Success(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	tarima := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}

	id, err := repo.Create(ctx, tarima)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id <= 0 {
		t.Errorf("Create returned id = %d, want > 0", id)
	}

	created, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID after Create: %v", err)
	}
	if created.CodigoBarras != tarima.CodigoBarras {
		t.Errorf("CodigoBarras = %q, want %q", created.CodigoBarras, tarima.CodigoBarras)
	}
	if created.NumeroProducto != "880197" {
		t.Errorf("NumeroProducto = %q, want 880197", created.NumeroProducto)
	}
	if created.Conservacion != "1" {
		t.Errorf("Conservacion = %q, want 1", created.Conservacion)
	}
}

func TestTarimaCreate_Duplicado(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	tarima1 := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	_, err := repo.Create(ctx, tarima1)
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	tarima2 := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	_, err = repo.Create(ctx, tarima2)
	if err == nil {
		t.Fatal("expected error for duplicate, got nil")
	}
}

func TestGetByID_Exists(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.NumeroProducto != "880197" {
		t.Errorf("NumeroProducto = %q, want 880197", got.NumeroProducto)
	}
	if got.NumeroTarima != "000999" {
		t.Errorf("NumeroTarima = %q, want 000999", got.NumeroTarima)
	}
	if got.NumeroUsuario != "010" {
		t.Errorf("NumeroUsuario = %q, want 010", got.NumeroUsuario)
	}
	if got.Conservacion != "1" {
		t.Errorf("Conservacion = %q, want 1", got.Conservacion)
	}
	if got.CantidadCajas != 45 {
		t.Errorf("CantidadCajas = %d, want 45", got.CantidadCajas)
	}
	if got.Peso != 450.00 {
		t.Errorf("Peso = %f, want 450.00", got.Peso)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent tarima, got nil")
	}
}

func TestUpdate_Success(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.ID = id
	created.NumeroTarima = "001001"
	created.CantidadCajas = 50
	created.Peso = 500.00
	created.Conservacion = "3"

	if err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if got.NumeroTarima != "001001" {
		t.Errorf("NumeroTarima = %q, want 001001", got.NumeroTarima)
	}
	if got.CantidadCajas != 50 {
		t.Errorf("CantidadCajas = %d, want 50", got.CantidadCajas)
	}
	if got.Peso != 500.00 {
		t.Errorf("Peso = %f, want 500.00", got.Peso)
	}
	if got.Conservacion != "3" {
		t.Errorf("Conservacion = %q, want 3", got.Conservacion)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	tt := &tarima.Tarima{
		ID:             99999,
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	err := repo.Update(ctx, tt)
	if err != tarima.ErrTarimaNoEncontrada {
		t.Errorf("got %v, want ErrTarimaNoEncontrada", err)
	}
}

func TestGetByIDRaw_Exists(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByIDRaw(ctx, id)
	if err != nil {
		t.Fatalf("GetByIDRaw: %v", err)
	}
	if got.NumeroProducto != "880197" {
		t.Errorf("NumeroProducto = %q, want 880197", got.NumeroProducto)
	}
	if got.NumeroTarima != "000999" {
		t.Errorf("NumeroTarima = %q, want 000999", got.NumeroTarima)
	}
}

func TestGetByIDRaw_NotFound(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	_, err := repo.GetByIDRaw(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent tarima, got nil")
	}
}

func TestDelete_Success(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	eliminada, err := repo.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if eliminada.ID != id {
		t.Errorf("eliminada.ID = %d, want %d", eliminada.ID, id)
	}
	if eliminada.NumeroProducto != "880197" {
		t.Errorf("NumeroProducto = %q, want 880197", eliminada.NumeroProducto)
	}

	_, err = repo.GetByIDRaw(ctx, id)
	if err == nil {
		t.Error("tarima should no longer exist after delete")
	}

	var historialCount int
	err = repo.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM historial_tarimas WHERE id_tarima_eliminada = ?`, id).Scan(&historialCount)
	if err != nil {
		t.Fatalf("contar historial: %v", err)
	}
	if historialCount != 1 {
		t.Errorf("historial count = %d, want 1", historialCount)
	}
}

func TestDelete_VerificaCamposHistorial(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	adminID := int64(1)
	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
		Descripcion:    "Tarima de prueba",
		IDUsuario:      &adminID,
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = repo.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var (
		hID             int64
		hCodigoBarras   string
		hNumeroProducto string
		hNumeroTarima   string
		hCantidadCajas  int
		hPeso           float64
		hNumeroVenta    string
		hDescripcion    string
		hIDUsuario      sql.NullInt64
	)
	err = repo.db.QueryRowContext(ctx, `
		SELECT id_tarima_eliminada, codigo_barras, numero_producto, numero_tarima,
		       cantidad_cajas, peso, numero_venta, descripcion, id_usuario
		FROM historial_tarimas WHERE id_tarima_eliminada = ?`, id).Scan(
		&hID, &hCodigoBarras, &hNumeroProducto, &hNumeroTarima,
		&hCantidadCajas, &hPeso, &hNumeroVenta, &hDescripcion, &hIDUsuario)
	if err != nil {
		t.Fatalf("leer historial: %v", err)
	}
	if hID != id {
		t.Errorf("id_tarima_eliminada = %d, want %d", hID, id)
	}
	if hCodigoBarras != created.CodigoBarras {
		t.Errorf("codigo_barras = %q, want %q", hCodigoBarras, created.CodigoBarras)
	}
	if hNumeroProducto != "880197" {
		t.Errorf("numero_producto = %q, want 880197", hNumeroProducto)
	}
	if hNumeroTarima != "000999" {
		t.Errorf("numero_tarima = %q, want 000999", hNumeroTarima)
	}
	if hCantidadCajas != 45 {
		t.Errorf("cantidad_cajas = %d, want 45", hCantidadCajas)
	}
	if hPeso != 450.00 {
		t.Errorf("peso = %f, want 450.00", hPeso)
	}
	if hNumeroVenta != "25-123456" {
		t.Errorf("numero_venta = %q, want 25-123456", hNumeroVenta)
	}
	if hDescripcion != "Tarima de prueba" {
		t.Errorf("descripcion = %q, want 'Tarima de prueba'", hDescripcion)
	}
	if !hIDUsuario.Valid || hIDUsuario.Int64 != adminID {
		t.Errorf("id_usuario = %v, want %d", hIDUsuario, adminID)
	}
}

func TestDelete_NotFound(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	_, err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent tarima, got nil")
	}

	var historialCount int
	err = repo.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM historial_tarimas`).Scan(&historialCount)
	if err != nil {
		t.Fatalf("contar historial: %v", err)
	}
	if historialCount != 0 {
		t.Errorf("historial should be empty, got %d", historialCount)
	}
}

func TestListHistorial_ReturnsEliminada(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	adminID := int64(1)
	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
		Descripcion:    "Tarima de prueba",
		IDUsuario:      &adminID,
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	eliminadas, err := repo.ListHistorial(ctx, tarima.FiltrosHistorial{}, 1000)
	if err != nil {
		t.Fatalf("ListHistorial: %v", err)
	}
	if len(eliminadas) != 1 {
		t.Fatalf("ListHistorial = %d registros, se esperaba 1", len(eliminadas))
	}

	h := eliminadas[0]
	if h.IDTarimaEliminada != id {
		t.Errorf("id_tarima_eliminada = %d, want %d", h.IDTarimaEliminada, id)
	}
	if h.NumeroProducto != "880197" {
		t.Errorf("numero_producto = %q, want 880197", h.NumeroProducto)
	}
	if h.NumeroTarima != "000999" {
		t.Errorf("numero_tarima = %q, want 000999", h.NumeroTarima)
	}
	if h.CantidadCajas != 45 {
		t.Errorf("cantidad_cajas = %d, want 45", h.CantidadCajas)
	}
	if h.Peso != 450.00 {
		t.Errorf("peso = %f, want 450.00", h.Peso)
	}
	if h.NumeroVenta != "25-123456" {
		t.Errorf("numero_venta = %q, want 25-123456", h.NumeroVenta)
	}
	if h.Descripcion != "Tarima de prueba" {
		t.Errorf("descripcion = %q, want 'Tarima de prueba'", h.Descripcion)
	}
	if h.IDUsuario == nil || *h.IDUsuario != adminID {
		t.Errorf("id_usuario = %v, want %d", h.IDUsuario, adminID)
	}
	if h.FechaEliminacion.IsZero() {
		t.Error("fecha_eliminacion no debería ser cero")
	}
}

func TestListHistorial_FiltraPorProducto(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	created1 := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	created2 := &tarima.Tarima{
		CodigoBarras:   "088019900099981020004545000000",
		NumeroProducto: "880199",
		NumeroTarima:   "000998",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  30,
		Peso:           300.00,
		NumeroVenta:    "25-123457",
	}

	id1, err := repo.Create(ctx, created1)
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	id2, err := repo.Create(ctx, created2)
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	if _, err := repo.Delete(ctx, id1); err != nil {
		t.Fatalf("Delete 1: %v", err)
	}
	if _, err := repo.Delete(ctx, id2); err != nil {
		t.Fatalf("Delete 2: %v", err)
	}

	eliminadas, err := repo.ListHistorial(ctx, tarima.FiltrosHistorial{
		FiltrosTarima: tarima.FiltrosTarima{NumeroProducto: "880197"},
	}, 1000)
	if err != nil {
		t.Fatalf("ListHistorial filtrado: %v", err)
	}
	if len(eliminadas) != 1 {
		t.Fatalf("ListHistorial filtrado = %d registros, se esperaba 1", len(eliminadas))
	}
	if eliminadas[0].IDTarimaEliminada != id1 {
		t.Errorf("filtrado devolvió la tarima %d, want %d", eliminadas[0].IDTarimaEliminada, id1)
	}
}

func TestListHistorial_FiltraPorFechaEliminacion(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	created := &tarima.Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	id, err := repo.Create(ctx, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	hoy := time.Now().Local().Format("2006-01-02")
	eliminadas, err := repo.ListHistorial(ctx, tarima.FiltrosHistorial{FechaEliminacion: hoy}, 1000)
	if err != nil {
		t.Fatalf("ListHistorial por fecha eliminación: %v", err)
	}
	if len(eliminadas) != 1 {
		t.Fatalf("ListHistorial por fecha = %d registros, se esperaba 1", len(eliminadas))
	}
}
