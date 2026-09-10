package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPort    = ":8080"
	DefaultDBPath  = "./data/gestiontarimas.db"
	DefaultTZ      = "America/Argentina/Buenos_Aires"
	DefaultAppName = "Gestión de Tarimas"
	minPort        = 1
	maxPort        = 65535
)

type Config struct {
	Port    string
	DBPath  string
	TZ      string
	AppName string
}

func Load() (Config, error) {
	cfg := Config{
		Port:    envOr("PORT", DefaultPort),
		DBPath:  envOr("DB_PATH", DefaultDBPath),
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
