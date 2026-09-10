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
	"strconv"
	"syscall"
	"time"

	"gestion-ciclo-tres/internal/config"
	"gestion-ciclo-tres/internal/delivery/handlers"
	"gestion-ciclo-tres/internal/delivery/middleware"
	"gestion-ciclo-tres/internal/delivery/render"
	"gestion-ciclo-tres/internal/identity"
	"gestion-ciclo-tres/internal/infrastructure/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed web/templates
var templatesFS embed.FS

//go:embed web/static
var staticFS embed.FS

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

	store := middleware.NewSessionStore()
	renderer, err := render.New(templatesFS)
	if err != nil {
		slog.Error("error al crear renderer", "error", err)
		os.Exit(1)
	}
	userRepo := sqlite.NewUserRepository(db)
	authSvc := identity.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(renderer, authSvc, store)

	srv := &http.Server{
		Addr:              cfg.Port,
		Handler:           routes(db, store, authHandler, cfg.AppName),
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

func routes(db *sql.DB, store *middleware.SessionStore, authHandler *handlers.AuthHandler, appName string) http.Handler {
	mux := http.NewServeMux()

	// Auth (públicos)
	mux.HandleFunc("GET /login", authHandler.ShowLogin)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	// Auth (protegidos: nivel 4)
	mux.Handle("GET /register",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 4)(
				http.HandlerFunc(authHandler.ShowRegister))))
	mux.Handle("POST /register",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 4)(
				http.HandlerFunc(authHandler.Register))))

	// Dashboard placeholder (protegido: nivel 1)
	mux.Handle("GET /",
		middleware.AuthRequired(store)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				session := middleware.SessionFromContext(r)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<!DOCTYPE html><html><head><title>` + appName + `</title>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
</head><body>
<div class="container my-5"><div class="row justify-content-center"><div class="col-md-8">
<div class="card"><div class="card-header bg-primary text-white"><h4><i class="fas fa-tachometer-alt me-2"></i>Dashboard</h4></div>
<div class="card-body">
<p>Bienvenido, <strong>` + session.Username + `</strong> (rol: ` + session.NombreRol + `, nivel: ` + strconv.Itoa(session.Nivel) + `)</p>
<p class="text-muted">Dashboard completo disponible en F5.</p>
<form hx-post="/logout" hx-swap="none"><button class="btn btn-danger"><i class="fas fa-sign-out-alt me-1"></i> Cerrar Sesión</button></form>
</div></div></div></div></div>
<script src="https://unpkg.com/htmx.org@2.0.4"></script>
</body></html>`))
			})))

	// Healthz (público)
	mux.Handle("GET /healthz", handlers.NewHealthz(db))

	// Static files
	staticSub, _ := fs.Sub(staticFS, "web/static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	return mux
}
