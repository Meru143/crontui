package config

import (
	"os"
	"path/filepath"
)

// Config holds application configuration.
type Config struct {
	MaxBackups   int    `mapstructure:"max_backups"`
	ShowNextRuns int    `mapstructure:"show_next_runs"`
	BackupDir    string `mapstructure:"backup_dir"`
	LogLevel     string `mapstructure:"log_level"`
	DateFormat   string `mapstructure:"date_format"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxBackups:   10,
		ShowNextRuns: 5,
		BackupDir:    defaultBackupDir(),
		LogLevel:     "info",
		DateFormat:   "2006-01-02 15:04:05",
	}
}

func defaultBackupDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "crontui", "backups")
}
