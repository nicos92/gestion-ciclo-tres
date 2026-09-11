package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("TZ", "")
	t.Setenv("APP_NAME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, want %q", cfg.Port, DefaultPort)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() error = %v", err)
	}
	wantDBPath := filepath.Join(dir, configSubdir, dbFileName)
	if cfg.DBPath != wantDBPath {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, wantDBPath)
	}
	if cfg.TZ != DefaultTZ {
		t.Errorf("TZ = %q, want %q", cfg.TZ, DefaultTZ)
	}
	if cfg.AppName != DefaultAppName {
		t.Errorf("AppName = %q, want %q", cfg.AppName, DefaultAppName)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9001")
	t.Setenv("DB_PATH", "/var/lib/db.db")
	t.Setenv("TZ", "UTC")
	t.Setenv("APP_NAME", "Test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != ":9001" {
		t.Errorf("Port = %q, want :9001", cfg.Port)
	}
	if cfg.DBPath != "/var/lib/db.db" {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
	if cfg.TZ != "UTC" {
		t.Errorf("TZ = %q", cfg.TZ)
	}
	if cfg.AppName != "Test" {
		t.Errorf("AppName = %q", cfg.AppName)
	}
}

func TestPortAlreadyPrefixed(t *testing.T) {
	t.Setenv("PORT", ":443")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != ":443" {
		t.Errorf("Port = %q, want :443", cfg.Port)
	}
}

func TestInvalidPort(t *testing.T) {
	for _, p := range []string{"abc", "0", "70000", ":abc"} {
		t.Run("port="+p, func(t *testing.T) {
			t.Setenv("PORT", p)
			if _, err := Load(); err == nil {
				t.Errorf("Load() sin error para PORT=%q, se esperaba error", p)
			}
		})
	}
}

func TestInvalidTZ(t *testing.T) {
	t.Setenv("TZ", "No/Existe")

	if _, err := Load(); err == nil {
		t.Error("Load() sin error para TZ inválida")
	}
}

func TestNormalizePort(t *testing.T) {
	if got, err := normalizePort("8080"); err != nil || strings.TrimPrefix(got, ":") != "8080" {
		t.Errorf("normalizePort(8080) = %q, %v", got, err)
	}
}
