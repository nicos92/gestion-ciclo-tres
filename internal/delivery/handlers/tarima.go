package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gestion-ciclo-tres/internal/delivery/middleware"
	"gestion-ciclo-tres/internal/delivery/render"
	"gestion-ciclo-tres/internal/tarima"
)

type TarimaHandler struct {
	renderer *render.Renderer
	tarima   *tarima.TarimaService
	sessions *middleware.SessionStore
}

func NewTarimaHandler(r *render.Renderer, ts *tarima.TarimaService, s *middleware.SessionStore) *TarimaHandler {
	return &TarimaHandler{renderer: r, tarima: ts, sessions: s}
}

type tarimasPageData struct {
	Title      string
	AppName    string
	Session    *middleware.SessionData
	Tarimas    []tarima.Tarima
	CountToday int
	Filters    tarima.FiltrosTarima
	ShowAll    bool
}

const appName = "Gestión de Tarimas"

func (h *TarimaHandler) ListarTarimas(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	filters := parseFilters(r)
	showAll := r.URL.Query().Get("all") == "1"

	data, err := h.loadTarimas(r, filters, showAll)
	if err != nil {
		http.Error(w, "Error interno al listar tarimas", http.StatusInternalServerError)
		return
	}
	data.Title = "Inventario de Tarimas"
	data.AppName = appName
	data.Session = session

	h.renderer.Render(w, "tarimas", data, http.StatusOK)
}

func (h *TarimaHandler) ListarTarimasFragment(w http.ResponseWriter, r *http.Request) {
	filters := parseFilters(r)
	showAll := r.URL.Query().Get("all") == "1"

	data, err := h.loadTarimas(r, filters, showAll)
	if err != nil {
		http.Error(w, "Error interno al listar tarimas", http.StatusInternalServerError)
		return
	}

	// hx-push-url: mantiene la URL canónica /tarimas?<filtros> para
	// imprimir/compartir (el fragmento no es la URL definitiva).
	if url := canonicalTarimasURL(filters, showAll); url != "" {
		w.Header().Set("HX-Push-Url", url)
	}

	h.renderer.RenderPartial(w, "tabla_tarimas", data, http.StatusOK)
}

func (h *TarimaHandler) loadTarimas(r *http.Request, filters tarima.FiltrosTarima, showAll bool) (*tarimasPageData, error) {
	ctx := r.Context()

	tarimas, err := h.tarima.List(ctx, filters, showAll, tarima.DefaultLimit)
	if err != nil {
		return nil, err
	}
	countToday, err := h.tarima.CountToday(ctx)
	if err != nil {
		return nil, err
	}

	return &tarimasPageData{
		Tarimas:    tarimas,
		CountToday: countToday,
		Filters:    filters,
		ShowAll:    showAll,
	}, nil
}

func parseFilters(r *http.Request) tarima.FiltrosTarima {
	q := r.URL.Query()
	f := tarima.FiltrosTarima{
		NumeroProducto: strings.TrimSpace(q.Get("numero_producto")),
		NumeroTarima:   strings.TrimSpace(q.Get("numero_tarima")),
		NumeroUsuario:  strings.TrimSpace(q.Get("numero_usuario")),
		NumeroVenta:    strings.TrimSpace(q.Get("numero_venta")),
		FechaRegistro:  strings.TrimSpace(q.Get("fecha_registro")),
		Legajo:         strings.TrimSpace(q.Get("legajo")),
		NombreUsuario:  strings.TrimSpace(q.Get("nombre_usuario")),
	}

	if v := strings.TrimSpace(q.Get("cantidad_cajas_min")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.CantidadCajasMin = &n
		}
	}
	if v := strings.TrimSpace(q.Get("peso_min")); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			f.PesoMin = &p
		}
	}
	return f
}

func canonicalTarimasURL(filters tarima.FiltrosTarima, showAll bool) string {
	vals := url.Values{}
	if filters.NumeroProducto != "" {
		vals.Set("numero_producto", filters.NumeroProducto)
	}
	if filters.NumeroTarima != "" {
		vals.Set("numero_tarima", filters.NumeroTarima)
	}
	if filters.NumeroUsuario != "" {
		vals.Set("numero_usuario", filters.NumeroUsuario)
	}
	if filters.NumeroVenta != "" {
		vals.Set("numero_venta", filters.NumeroVenta)
	}
	if filters.FechaRegistro != "" {
		vals.Set("fecha_registro", filters.FechaRegistro)
	}
	if filters.Legajo != "" {
		vals.Set("legajo", filters.Legajo)
	}
	if filters.NombreUsuario != "" {
		vals.Set("nombre_usuario", filters.NombreUsuario)
	}
	if filters.CantidadCajasMin != nil {
		vals.Set("cantidad_cajas_min", strconv.Itoa(*filters.CantidadCajasMin))
	}
	if filters.PesoMin != nil {
		vals.Set("peso_min", strconv.FormatFloat(*filters.PesoMin, 'f', -1, 64))
	}
	if showAll {
		vals.Set("all", "1")
	}
	if len(vals) == 0 {
		return ""
	}
	return "/tarimas?" + vals.Encode()
}