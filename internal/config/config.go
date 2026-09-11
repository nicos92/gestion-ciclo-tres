package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPort    = ":8080"
	DefaultTZ      = "America/Argentina/Buenos_Aires"
	DefaultAppName = "Gestión de Tarimas"
	minPort        = 1
	maxPort        = 65535
	configSubdir   = "nicolas-sandoval/gestion-ciclo-tres"
	dbFileName     = "gestion-ciclo-tres.db"
)

type Config struct {
	Port    string
	DBPath  string
	TZ      string
	AppName string
}

func Load() (Config, error) {
	dbPath, err := defaultDBPath()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:    envOr("PORT", DefaultPort),
		DBPath:  envOr("DB_PATH", dbPath),
		TZ:      envOr("TZ", DefaultTZ),
		AppName: envOr("APP_NAME", DefaultAppName),
	}

	port, err := normalizePort(cfg.Port)
	if err != nil {
		return Config{}, err
	}
	cfg.Port = port

	if _, err := time.LoadLocation(cfg.TZ); err != nil {
		return Config{}, fmt.Errorf("TZ inválida %q: %w", cfg.TZ, err)
	}

	return cfg, nil
}

// defaultDBPath devuelve la ruta de la base de datos en el directorio de
// configuración del usuario (Linux ~/.config, Windows %APPDATA%).
func defaultDBPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no se pudo determinar el directorio de configuración: %w", err)
	}
	return filepath.Join(dir, configSubdir, dbFileName), nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func normalizePort(port string) (string, error) {
	p := strings.TrimPrefix(port, ":")
	if p == "" {
		return "", fmt.Errorf("puerto vacío")
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return "", fmt.Errorf("puerto %q inválido: %w", port, err)
	}
	if n < minPort || n > maxPort {
		return "", fmt.Errorf("puerto %d fuera de rango (%d-%d)", n, minPort, maxPort)
	}
	return ":" + p, nil
}
