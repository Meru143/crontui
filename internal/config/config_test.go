package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxBackups != 10 {
		t.Errorf("MaxBackups = %d, want 10", cfg.MaxBackups)
	}
	if cfg.ShowNextRuns != 5 {
		t.Errorf("ShowNextRuns = %d, want 5", cfg.ShowNextRuns)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.DateFormat != "2006-01-02 15:04:05" {
		t.Errorf("DateFormat = %q, want %q", cfg.DateFormat, "2006-01-02 15:04:05")
	}
	if cfg.BackupDir == "" {
		t.Error("BackupDir should not be empty")
	}
}

func TestDefaultBackupDir(t *testing.T) {
	dir := defaultBackupDir()
	if dir == "" {
		t.Error("defaultBackupDir should return non-empty string")
	}

	home, err := os.UserHomeDir()
	if err == nil {
		want := filepath.Join(home, ".config", "crontui", "backups")
		if dir != want {
			t.Errorf("defaultBackupDir = %q, want %q", dir, want)
		}
	}
}

func TestLoad_FromConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "max_backups": 3,
  "show_next_runs": 9,
  "backup_dir": "/tmp/crontui-backups",
  "log_level": "debug"
}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CRONTUI_CONFIG", configPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.MaxBackups != 3 {
		t.Fatalf("MaxBackups = %d, want 3", cfg.MaxBackups)
	}
	if cfg.ShowNextRuns != 9 {
		t.Fatalf("ShowNextRuns = %d, want 9", cfg.ShowNextRuns)
	}
	if cfg.BackupDir != "/tmp/crontui-backups" {
		t.Fatalf("BackupDir = %q, want %q", cfg.BackupDir, "/tmp/crontui-backups")
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.DateFormat != "2006-01-02 15:04:05" {
		t.Fatalf("DateFormat = %q, want default format", cfg.DateFormat)
	}
}

func TestLoad_EnvOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"max_backups": 3, "log_level": "warn"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CRONTUI_CONFIG", configPath)
	t.Setenv("CRONTUI_MAX_BACKUPS", "12")
	t.Setenv("CRONTUI_BACKUP_DIR", "/var/lib/crontui/backups")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.MaxBackups != 12 {
		t.Fatalf("MaxBackups = %d, want 12", cfg.MaxBackups)
	}
	if cfg.BackupDir != "/var/lib/crontui/backups" {
		t.Fatalf("BackupDir = %q, want %q", cfg.BackupDir, "/var/lib/crontui/backups")
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{invalid json`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("CRONTUI_CONFIG", configPath)

	if _, err := Load(); err == nil {
		t.Fatal("Load should fail for invalid config file")
	}
}

func TestLoad_InvalidEnvOverride(t *testing.T) {
	t.Setenv("CRONTUI_MAX_BACKUPS", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("Load should fail for invalid env override")
	}
}
