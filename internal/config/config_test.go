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
