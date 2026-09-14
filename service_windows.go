//go:build windows

package main

import (
	"log/slog"
	"os"

	"golang.org/x/sys/windows/svc"

	"gestion-ciclo-tres/internal/config"
)

const serviceName = "gestion-ciclo-tres"

// useWindowsService devuelve true si el proceso fue lanzado por el
// Administrador de servicios de Windows (SCM).
func useWindowsService() bool {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		slog.Warn("no se pudo determinar si el proceso corre como servicio", "error", err)
		return false
	}
	return isSvc
}

// runAsService registra la app como servicio nativo de Windows y corre hasta
// que el Administrador de servicios envíe un comando Stop o Shutdown.
func runAsService() {
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

	stopCh := make(chan struct{})
	appErr := make(chan error, 1)
	go func() {
		appErr <- run(stopCh, cfg)
	}()

	if err := svc.Run(serviceName, &svcHandler{stop: stopCh}); err != nil {
		slog.Error("servicio", "error", err)
		os.Exit(1)
	}

	slog.Info("servicio detenido")
	if err := <-appErr; err != nil {
		slog.Error("aplicación", "error", err)
	}
}

// svcHandler implementa svc.Handler para integrarse con el SCM de Windows.
type svcHandler struct {
	stop chan<- struct{}
}

func (h *svcHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			close(h.stop)
			return false, 0
		}
	}
	return false, 0
}