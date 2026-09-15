package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io"
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

const (
	NivelProduccion     = 1
	NivelSupervisor     = 2
	NivelJefeProduccion = 3
	NivelAdmin          = 4
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	if useWindowsService() {
		runAsService()
		return
	}

	runAsProcess()
}

// runAsProcess ejecuta la app como proceso normal, deteniéndose ante SIGINT/SIGTERM.
func runAsProcess() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	closeLogger, err := setupLogger(cfg)
	if err != nil {
		slog.Error("no se pudo configurar el archivo de log", "error", err)
		os.Exit(1)
	}
	defer closeLogger()

	ctxStop, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctxStop.Done(), cfg); err != nil {
		slog.Error("aplicación", "error", err)
		os.Exit(1)
	}
	slog.Info("servidor detenido")
}

// setupLogger configura slog hacia stdout y, si LOG_FILE está seteada,
// además escribe a ese archivo.
func setupLogger(cfg config.Config) (closeFunc func(), err error) {
	target := io.Writer(os.Stdout)
	var file *os.File

	if cfg.LogFile != "" {
		file, err = os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		target = io.MultiWriter(os.Stdout, file)
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(target, nil)))

	return func() {
		if file != nil {
			_ = file.Close()
		}
	}, nil
}

// run arranca el servidor HTTP y bloquea hasta que el servidor falle o el
// canal stop se cierre (señal del sistema en modo proceso, comando Stop del
// Administrador de servicios en modo Windows service).
func run(stop <-chan struct{}, cfg config.Config) error {
	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		return err
	}
	time.Local = loc

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	if err := sqlite.Migrate(ctx, db, migrationsDir); err != nil {
		return err
	}
	if err := sqlite.Seed(ctx, db); err != nil {
		return err
	}

	store := middleware.NewSessionStore()
	renderer, err := render.New(templatesFS)
	if err != nil {
		return err
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

	errCh := make(chan error, 1)
	go func() {
		slog.Info("servidor iniciado", "addr", cfg.Port, "app", cfg.AppName, "db", cfg.DBPath)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-stop:
		slog.Info("señal de apagado recibida")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func routes(
	db *sql.DB,
	store *middleware.SessionStore,
	authHandler *handlers.AuthHandler,
	tarimaHandler *handlers.TarimaHandler,
	usuarioHandler *handlers.UsuarioHandler,
) http.Handler {

	mux := http.NewServeMux()

	requireAuth := func(h http.Handler) http.Handler {
		return middleware.AuthRequired(store)(h)
	}

	requireNivel := func(nivel int, h http.Handler) http.Handler {
		return middleware.AuthRequired(store)(
			middleware.NivelRequerido(store, nivel)(h),
		)
	}

	// ============================================================
	// AUTH
	// ============================================================

	mux.HandleFunc("GET /login", authHandler.ShowLogin)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	mux.Handle("GET /register",
		requireNivel(
			NivelAdmin,
			http.HandlerFunc(authHandler.ShowRegister),
		),
	)

	mux.Handle("POST /register",
		requireNivel(
			NivelAdmin,
			http.HandlerFunc(authHandler.Register),
		),
	)

	// ============================================================
	// DASHBOARD
	// ============================================================

	mux.Handle("GET /dashboard",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(usuarioHandler.ShowDashboard),
		),
	)

	mux.Handle("GET /",
		requireAuth(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/dashboard", http.StatusFound)
			}),
		),
	)

	// ============================================================
	// TARIMAS - CONSULTA
	// ============================================================

	mux.Handle("GET /tarimas",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(tarimaHandler.ListarTarimas),
		),
	)

	mux.Handle("GET /tarimas/lista",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(tarimaHandler.ListarTarimasFragment),
		),
	)

	mux.Handle("GET /tarimas/historial",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(tarimaHandler.ListarHistorial),
		),
	)

	mux.Handle("GET /tarimas/historial/lista",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(tarimaHandler.ListarHistorialFragment),
		),
	)

	// ============================================================
	// TARIMAS - CREACIÓN
	// ============================================================

	mux.Handle("GET /tarimas/nueva",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(tarimaHandler.ShowNuevaTarima),
		),
	)

	mux.Handle("POST /tarimas",
		requireNivel(
			NivelProduccion,
			http.HandlerFunc(tarimaHandler.GuardarTarima),
		),
	)

	// ============================================================
	// TARIMAS - EDICIÓN
	// ============================================================

	mux.Handle("GET /tarimas/editar/{id}",
		requireNivel(
			NivelSupervisor,
			http.HandlerFunc(tarimaHandler.ShowEditarTarima),
		),
	)

	mux.Handle("POST /tarimas/actualizar/{id}",
		requireNivel(
			NivelSupervisor,
			http.HandlerFunc(tarimaHandler.ActualizarTarima),
		),
	)

	// ============================================================
	// TARIMAS - ELIMINACIÓN
	// ============================================================

	mux.Handle("DELETE /tarimas/{id}",
		requireNivel(
			NivelSupervisor,
			http.HandlerFunc(tarimaHandler.EliminarTarima),
		),
	)

	// ============================================================
	// USUARIOS
	// ============================================================

	mux.Handle("GET /usuarios",
		requireNivel(
			NivelAdmin,
			http.HandlerFunc(usuarioHandler.ListarUsuarios),
		),
	)

	mux.Handle("GET /usuarios/editar/{id}",
		requireNivel(
			NivelAdmin,
			http.HandlerFunc(usuarioHandler.ShowEditarUsuario),
		),
	)

	mux.Handle("POST /usuarios/actualizar/{id}",
		requireNivel(
			NivelAdmin,
			http.HandlerFunc(usuarioHandler.ActualizarUsuario),
		),
	)

	// ============================================================
	// HEALTH CHECK
	// ============================================================

	mux.Handle(
		"GET /healthz",
		handlers.NewHealthz(db),
	)

	// ============================================================
	// STATIC
	// ============================================================

	staticSub, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		panic(err)
	}

	mux.Handle(
		"GET /static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.FS(staticSub)),
		),
	)

	return mux
}
