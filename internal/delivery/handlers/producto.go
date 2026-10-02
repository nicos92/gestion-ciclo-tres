package handlers

import (
	"net/http"
	"strings"

	"gestion-ciclo-tres/internal/delivery/render"
	"gestion-ciclo-tres/internal/producto"
)

type ProductoHandler struct {
	renderer *render.Renderer
	catalogo *producto.Catalogo
}

func NewProductoHandler(r *render.Renderer, c *producto.Catalogo) *ProductoHandler {
	return &ProductoHandler{renderer: r, catalogo: c}
}

// campoProductoData alimenta el partial campo_nombre_producto.
type campoProductoData struct {
	Numero string
	Nombre string
}

// Buscar resuelve el número de producto contra el catálogo en memoria y
// devuelve el campo de sólo lectura ya renderizado. Responde 200 tanto si el
// producto existe como si no, para que el swap sea siempre válido.
func (h *ProductoHandler) Buscar(w http.ResponseWriter, r *http.Request) {
	numero := strings.TrimSpace(r.URL.Query().Get("numeroProducto"))

	data := campoProductoData{Numero: numero}
	if p, ok := h.catalogo.Buscar(numero); ok {
		data.Nombre = p.Nombre
	}

	h.renderer.RenderPartial(w, "campo_nombre_producto", data, http.StatusOK)
}
