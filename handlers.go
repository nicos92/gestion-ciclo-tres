package main

import (
	"errors"
	"net/http"
	"strconv"
)

type todoHandlers struct {
	service *todoService
}

func newTodoHandlers(service *todoService) *todoHandlers {
	return &todoHandlers{service: service}
}

func (h *todoHandlers) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("POST /todo", h.create)
	mux.HandleFunc("DELETE /todo/{id}", h.delete)
	mux.HandleFunc("POST /todo/{id}/toggle", h.toggle)
}

func (h *todoHandlers) index(w http.ResponseWriter, r *http.Request) {
	render(w, "index", h.service.List())
}

func (h *todoHandlers) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, err := h.service.Create(r.FormValue("text"))
	if err != nil {
		if errors.Is(err, ErrEmptyText) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	render(w, "todo_item", item)
}

func (h *todoHandlers) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *todoHandlers) toggle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	item, err := h.service.Toggle(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	render(w, "todo_item", item)
}