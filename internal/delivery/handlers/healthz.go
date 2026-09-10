package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type Healthz struct {
	db *sql.DB
}

func NewHealthz(db *sql.DB) *Healthz {
	return &Healthz{db: db}
}

// GET /healthz: responde 200 solo si la DB responde al ping; si no, 503.
func (h *Healthz) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
