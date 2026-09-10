package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gestion-ciclo-tres/internal/config"
	"gestion-ciclo-tres/internal/delivery/handlers"
	"gestion-ciclo-tres/internal/infrastructure/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		slog.Error("zona horaria inválida", "tz", cfg.TZ, "error", err)
		os.Exit(1)
	}
	time.Local = loc

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		slog.Error("no se pudo abrir la base de datos", "db", cfg.DBPath, "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		slog.Error("preparar migraciones", "error", err)
		os.Exit(1)
	}
	if err := sqlite.Migrate(ctx, db, migrationsDir); err != nil {
		slog.Error("error en migraciones", "error", err)
		os.Exit(1)
	}
	if err := sqlite.Seed(ctx, db); err != nil {
		slog.Error("error en seed", "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.Port,
		Handler:           routes(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctxStop, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("servidor iniciado", "addr", cfg.Port, "app", cfg.AppName, "db", cfg.DBPath)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("servidor", "error", err)
			os.Exit(1)
		}
	case <-ctxStop.Done():
		slog.Info("señal de apagado recibida")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("servidor detenido")
}

func routes(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", handlers.NewHealthz(db))
	return mux
}
