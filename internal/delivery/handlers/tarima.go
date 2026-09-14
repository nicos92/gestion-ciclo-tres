package handlers

import (
	"errors"
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

type tarimaFormData struct {
	Title    string
	AppName  string
	Session  *middleware.SessionData
	Tarima   *tarima.Tarima
	Error    string
	Success  string
	EditMode bool
}

const appName = "Gestión Ciclo Tres"

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

	if url := canonicalTarimasURL(filters, showAll); url != "" {
		w.Header().Set("HX-Push-Url", url)
	}

	h.renderer.RenderPartial(w, "tabla_tarimas", data, http.StatusOK)
}

func (h *TarimaHandler) ShowNuevaTarima(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)
	data := tarimaFormData{
		Title:    "Nueva Tarima",
		AppName:  appName,
		Session:  session,
		EditMode: false,
	}
	h.renderer.Render(w, "nueva_tarima", data, http.StatusOK)
}

func (h *TarimaHandler) GuardarTarima(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, session, nil, "", "validation", false)
		return
	}

	t := tarima.Tarima{
		CodigoBarras:   r.FormValue("codigoBarras"),
		NumeroProducto: r.FormValue("numeroProducto"),
		NumeroTarima:   r.FormValue("numeroTarima"),
		NumeroUsuario:  r.FormValue("numeroUsuario"),
		Conservacion:   r.FormValue("conservacion"),
		NumeroVenta:    r.FormValue("numeroVenta"),
		Descripcion:    r.FormValue("descripcion"),
	}

	if v := r.FormValue("cantidadCajas"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.CantidadCajas = n
		}
	}
	if v := r.FormValue("peso"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			t.Peso = p
		}
	}

	session2 := middleware.SessionFromContext(r)
	if session2 != nil {
		t.IDUsuario = &session2.UserID
	}

	_, err := h.tarima.Create(r.Context(), &t)
	if err != nil {
		h.renderFormError(w, session, &t, tarimaErrorKey(err), "", false)
		return
	}

	h.renderFormSuccess(w, session, "created", false)
}

func (h *TarimaHandler) ShowEditarTarima(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/tarimas", http.StatusFound)
		return
	}

	t, err := h.tarima.GetByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/tarimas", http.StatusFound)
		return
	}

	data := tarimaFormData{
		Title:    "Editar Tarima",
		AppName:  appName,
		Session:  session,
		Tarima:   t,
		EditMode: true,
	}
	h.renderer.Render(w, "editar_tarima", data, http.StatusOK)
}

func (h *TarimaHandler) ActualizarTarima(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/tarimas", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, session, nil, "", "validation", true)
		return
	}

	t := tarima.Tarima{
		ID:             id,
		CodigoBarras:   r.FormValue("codigoBarras"),
		NumeroProducto: r.FormValue("numeroProducto"),
		NumeroTarima:   r.FormValue("numeroTarima"),
		NumeroUsuario:  r.FormValue("numeroUsuario"),
		Conservacion:   r.FormValue("conservacion"),
		NumeroVenta:    r.FormValue("numeroVenta"),
		Descripcion:    r.FormValue("descripcion"),
	}

	if v := r.FormValue("cantidadCajas"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.CantidadCajas = n
		}
	}
	if v := r.FormValue("peso"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			t.Peso = p
		}
	}

	if err := h.tarima.Update(r.Context(), &t); err != nil {
		data := tarimaFormData{
			Title:    "Editar Tarima",
			AppName:  appName,
			Session:  session,
			Tarima:   &t,
			Error:    tarimaErrorKey(err),
			EditMode: true,
		}
		h.renderer.RenderPartial(w, "form_tarimas", data, http.StatusOK)
		return
	}

	updated, _ := h.tarima.GetByID(r.Context(), id)
	data := tarimaFormData{
		Title:    "Editar Tarima",
		AppName:  appName,
		Session:  session,
		Tarima:   updated,
		Success:  "updated",
		EditMode: true,
	}
	h.renderer.RenderPartial(w, "form_tarimas", data, http.StatusOK)
}

func (h *TarimaHandler) EliminarTarima(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	_, err = h.tarima.Delete(r.Context(), id)
	if err != nil {
		if err == tarima.ErrTarimaNoEncontrada {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func tarimaErrorKey(err error) string {
	switch {
	case errors.Is(err, tarima.ErrCodigoBarrasDuplicado):
		return "duplicate"
	case errors.Is(err, tarima.ErrBarcodeLongitud):
		return "barcode_length"
	case errors.Is(err, tarima.ErrBarcodePrefix):
		return "barcode_prefix"
	case errors.Is(err, tarima.ErrBarcodeMarker):
		return "barcode_marker"
	case errors.Is(err, tarima.ErrBarcodeNoCoincide):
		return "barcode_mismatch"
	case errors.Is(err, tarima.ErrTarimaNoEncontrada):
		return "not_found"
	default:
		return "validation"
	}
}

func (h *TarimaHandler) renderFormError(w http.ResponseWriter, session *middleware.SessionData, t *tarima.Tarima, errKey, fallbackErr string, editMode bool) {
	key := errKey
	if key == "" {
		key = fallbackErr
	}
	data := tarimaFormData{
		Title:    "Nueva Tarima",
		AppName:  appName,
		Session:  session,
		Tarima:   t,
		Error:    key,
		EditMode: editMode,
	}
	h.renderer.RenderPartial(w, "form_tarimas", data, http.StatusOK)
}

func (h *TarimaHandler) renderFormSuccess(w http.ResponseWriter, session *middleware.SessionData, successKey string, editMode bool) {
	data := tarimaFormData{
		Title:    "Nueva Tarima",
		AppName:  appName,
		Session:  session,
		Success:  successKey,
		EditMode: editMode,
	}
	h.renderer.RenderPartial(w, "form_tarimas", data, http.StatusOK)
	w.Header().Set("HX-Push-Url", "/tarimas/nueva")
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
