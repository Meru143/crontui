package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds application configuration.
type Config struct {
	MaxBackups      int    `json:"max_backups" mapstructure:"max_backups"`
	ShowNextRuns    int    `json:"show_next_runs" mapstructure:"show_next_runs"`
	BackupDir       string `json:"backup_dir" mapstructure:"backup_dir"`
	LogLevel        string `json:"log_level" mapstructure:"log_level"`
	DateFormat      string `json:"date_format" mapstructure:"date_format"`
	WindowsTaskPath string `json:"windows_task_path" mapstructure:"windows_task_path"`
}

type configOverrides struct {
	MaxBackups      *int    `json:"max_backups"`
	ShowNextRuns    *int    `json:"show_next_runs"`
	BackupDir       *string `json:"backup_dir"`
	LogLevel        *string `json:"log_level"`
	DateFormat      *string `json:"date_format"`
	WindowsTaskPath *string `json:"windows_task_path"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxBackups:      10,
		ShowNextRuns:    5,
		BackupDir:       defaultBackupDir(),
		LogLevel:        "info",
		DateFormat:      "2006-01-02 15:04:05",
		WindowsTaskPath: `\CronTUI\`,
	}
}

// Load merges defaults with an optional config file and CRONTUI_ env vars.
func Load() (Config, error) {
	cfg := DefaultConfig()

	if err := loadConfigFile(&cfg); err != nil {
		return Config{}, err
	}
	if err := applyEnvOverrides(&cfg); err != nil {
		return Config{}, err
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultBackupDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "crontui", "backups")
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "crontui", "config.json")
}

func loadConfigFile(cfg *Config) error {
	path := os.Getenv("CRONTUI_CONFIG")
	if path == "" {
		path = defaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	var overrides configOverrides
	if err := json.Unmarshal(data, &overrides); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}

	applyOverrides(cfg, overrides)
	return nil
}

func applyEnvOverrides(cfg *Config) error {
	intOverrides := []struct {
		env string
		dst *int
	}{
		{env: "CRONTUI_MAX_BACKUPS", dst: &cfg.MaxBackups},
		{env: "CRONTUI_SHOW_NEXT_RUNS", dst: &cfg.ShowNextRuns},
	}

	for _, override := range intOverrides {
		value, ok := os.LookupEnv(override.env)
		if !ok || value == "" {
			continue
		}

		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse %s: %w", override.env, err)
		}
		*override.dst = n
	}

	stringOverrides := []struct {
		env string
		dst *string
	}{
		{env: "CRONTUI_BACKUP_DIR", dst: &cfg.BackupDir},
		{env: "CRONTUI_LOG_LEVEL", dst: &cfg.LogLevel},
		{env: "CRONTUI_DATE_FORMAT", dst: &cfg.DateFormat},
		{env: "CRONTUI_WINDOWS_TASK_PATH", dst: &cfg.WindowsTaskPath},
	}

	for _, override := range stringOverrides {
		value, ok := os.LookupEnv(override.env)
		if !ok || value == "" {
			continue
		}
		*override.dst = value
	}

	return nil
}

func applyOverrides(cfg *Config, overrides configOverrides) {
	if overrides.MaxBackups != nil {
		cfg.MaxBackups = *overrides.MaxBackups
	}
	if overrides.ShowNextRuns != nil {
		cfg.ShowNextRuns = *overrides.ShowNextRuns
	}
	if overrides.BackupDir != nil {
		cfg.BackupDir = *overrides.BackupDir
	}
	if overrides.LogLevel != nil {
		cfg.LogLevel = *overrides.LogLevel
	}
	if overrides.DateFormat != nil {
		cfg.DateFormat = *overrides.DateFormat
	}
	if overrides.WindowsTaskPath != nil {
		cfg.WindowsTaskPath = *overrides.WindowsTaskPath
	}
}

func validate(cfg Config) error {
	switch {
	case cfg.MaxBackups < 0:
		return fmt.Errorf("max_backups must be >= 0")
	case cfg.ShowNextRuns <= 0:
		return fmt.Errorf("show_next_runs must be > 0")
	case cfg.BackupDir == "":
		return fmt.Errorf("backup_dir must not be empty")
	case cfg.LogLevel == "":
		return fmt.Errorf("log_level must not be empty")
	case cfg.DateFormat == "":
		return fmt.Errorf("date_format must not be empty")
	case cfg.WindowsTaskPath == "":
		return fmt.Errorf("windows_task_path must not be empty")
	default:
		return nil
	}
}
