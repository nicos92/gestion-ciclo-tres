package sqlite

import (
	"context"
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

func (r *SQLiteTarimaRepository) insertTestTarima(ctx context.Context, t *testing.T, numeroProducto, numeroTarima, numeroUsuario string, cajas int, peso float64, numeroVenta string, idUsuario *int64, fechaRegistro string) {
	t.Helper()
	codigo, err := buildBarcode(numeroProducto, numeroTarima, numeroUsuario, cajas, peso)
	if err != nil {
		t.Fatalf("buildBarcode: %v", err)
	}
	venta := numeroVenta
	if venta == "" {
		venta = "VENTA-X"
	}

	stmt := `INSERT INTO tarimas (codigo_barras, numero_producto, numero_tarima, numero_usuario,
	               cantidad_cajas, peso, numero_venta, id_usuario, fecha_registro)
	       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, stmt,
		codigo, numeroProducto, numeroTarima, numeroUsuario, cajas, peso, venta, idUsuario, fechaRegistro); err != nil {
		t.Fatalf("insertar tarima de test: %v", err)
	}
}

func TestListToday_OnlyToday(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	ayer := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "100000", "100001", "50", 10, 100.5, "25-000001", nil, ayer)

	tarimas, err := repo.ListToday(ctx, 1000)
	if err != nil {
		t.Fatalf("ListToday: %v", err)
	}
	if len(tarimas) != 2 {
		t.Errorf("ListToday = %d tarimas, se esperaban 2 (los del seed)", len(tarimas))
	}
}

func TestListAll_ReturnsAll(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	ayer := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	adminID := int64(1)
	repo.insertTestTarima(ctx, t, "200000", "200001", "60", 5, 50.0, "25-000002", &adminID, ayer)

	tarimas, err := repo.ListAll(ctx, 1000)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(tarimas) != 3 {
		t.Errorf("ListAll = %d tarimas, se esperaban 3", len(tarimas))
	}
}

func TestListFiltered_ByProducto(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)
	adminID := int64(1)
	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "777888", "300001", "80", 12, 80.0, "25-000003", &adminID, hoy)

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
	repo.insertTestTarima(ctx, t, "111222", "400001", "90", 20, 200.0, "25-000004", &adminID, ayer)

	// El admin del seed tiene legajo "A0001".
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
	repo.insertTestTarima(ctx, t, "333444", "500001", "10", 10, 10.0, "25-000005", nil, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{CantidadCajasMin: &minCajas}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by cajas min: %v", err)
	}
	// Las 2 del seed tienen 45 y 30 cajas; la nueva tiene 10.
	if len(tarimas) != 1 {
		t.Errorf("ListFiltered cajas>=40 = %d, se esperaba 1", len(tarimas))
	}
	if tarimas[0].CantidadCajas != 45 {
		t.Errorf("CantidadCajas = %d, want 45", tarimas[0].CantidadCajas)
	}
}

func TestListFiltered_ByPesoMin(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)
	minPeso := 400.0

	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")
	repo.insertTestTarima(ctx, t, "555666", "600001", "20", 60, 600.0, "25-000006", nil, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{PesoMin: &minPeso}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by peso min: %v", err)
	}
	// Seed: 450 y 300; nueva: 600.
	if len(tarimas) != 2 {
		t.Errorf("ListFiltered peso>=400 = %d, se esperaba 2", len(tarimas))
	}
}

func TestListFiltered_ByFecha(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	ayer := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	ayerUTC := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05")
	hoy := time.Now().UTC().Format("2006-01-02 15:04:05")

	repo.insertTestTarima(ctx, t, "777111", "700001", "30", 99, 999.0, "25-000007", nil, hoy)

	tarimas, err := repo.ListFiltered(ctx, tarima.FiltrosTarima{FechaRegistro: ayer}, 1000)
	if err != nil {
		t.Fatalf("ListFiltered by fecha: %v", err)
	}
	if len(tarimas) != 0 {
		t.Errorf("fecha %s = %d tarimas, se esperaba 0 (la única de ayer está fuera de hoy)", ayer, len(tarimas))
	}

	// Insertar una con fecha de ayer y verificar que el filtro la encuentra.
	repo.insertTestTarima(ctx, t, "777222", "700002", "31", 5, 5.0, "25-000008", nil, ayerUTC)

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

	n, err := repo.CountToday(ctx)
	if err != nil {
		t.Fatalf("CountToday: %v", err)
	}
	if n != 2 {
		t.Errorf("CountToday = %d, se esperaban 2 (del seed)", n)
	}
}

func TestListLimit(t *testing.T) {
	ctx, repo := newTestTarimaRepo(t)

	// Insertar 3 tarimas extra de hoy para tener 5 en total.
	for i := 0; i < 3; i++ {
		repo.insertTestTarima(ctx, t, "111000", fmt.Sprintf("8000%02d", i), "11", 1, 1.0, "25-000009", nil,
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