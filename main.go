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
	"gestion-ciclo-tres/internal/delivery/middleware"
	"gestion-ciclo-tres/internal/delivery/render"
	"gestion-ciclo-tres/internal/identity"
	"gestion-ciclo-tres/internal/infrastructure/sqlite"
	"gestion-ciclo-tres/internal/tarima"
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

	tarimaRepo := sqlite.NewTarimaRepository(db)
	tarimaSvc := tarima.NewTarimaService(tarimaRepo)
	tarimaHandler := handlers.NewTarimaHandler(renderer, tarimaSvc, store)

	usuarioHandler := handlers.NewUsuarioHandler(renderer, authSvc, tarimaSvc, store)

	srv := &http.Server{
		Addr:              cfg.Port,
		Handler:           routes(db, store, authHandler, tarimaHandler, usuarioHandler),
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

func routes(db *sql.DB, store *middleware.SessionStore, authHandler *handlers.AuthHandler, tarimaHandler *handlers.TarimaHandler, usuarioHandler *handlers.UsuarioHandler) http.Handler {
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

	// Tarimas (protegidos: nivel 1)
	mux.Handle("GET /tarimas",
		middleware.AuthRequired(store)(
			http.HandlerFunc(tarimaHandler.ListarTarimas)))
	mux.Handle("GET /tarimas/lista",
		middleware.AuthRequired(store)(
			http.HandlerFunc(tarimaHandler.ListarTarimasFragment)))

	// Tarimas — alta (nivel 1)
	mux.Handle("GET /tarimas/nueva",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 1)(
				http.HandlerFunc(tarimaHandler.ShowNuevaTarima))))
	mux.Handle("POST /tarimas",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 1)(
				http.HandlerFunc(tarimaHandler.GuardarTarima))))

	// Tarimas — edición (nivel 2)
	mux.Handle("GET /tarimas/editar/{id}",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 2)(
				http.HandlerFunc(tarimaHandler.ShowEditarTarima))))
	mux.Handle("POST /tarimas/actualizar/{id}",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 2)(
				http.HandlerFunc(tarimaHandler.ActualizarTarima))))

	// Tarimas — eliminación (nivel 2: supervisor)
	mux.Handle("DELETE /tarimas/{id}",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 3)(
				http.HandlerFunc(tarimaHandler.EliminarTarima))))

	// Dashboard (protegido: nivel 1)
	mux.Handle("GET /dashboard",
		middleware.AuthRequired(store)(
			http.HandlerFunc(usuarioHandler.ShowDashboard)))

	// Usuarios — listar (nivel 4)
	mux.Handle("GET /usuarios",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 4)(
				http.HandlerFunc(usuarioHandler.ListarUsuarios))))

	// Usuarios — editar form (nivel 4)
	mux.Handle("GET /usuarios/editar/{id}",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 4)(
				http.HandlerFunc(usuarioHandler.ShowEditarUsuario))))

	// Usuarios — actualizar (nivel 4)
	mux.Handle("POST /usuarios/actualizar/{id}",
		middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, 4)(
				http.HandlerFunc(usuarioHandler.ActualizarUsuario))))

	// Home — redirect a dashboard
	mux.Handle("GET /",
		middleware.AuthRequired(store)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/dashboard", http.StatusFound)
			})))

	// Healthz (público)
	mux.Handle("GET /healthz", handlers.NewHealthz(db))

	// Static files
	staticSub, _ := fs.Sub(staticFS, "web/static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	return mux
}
