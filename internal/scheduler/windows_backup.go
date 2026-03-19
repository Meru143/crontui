package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

const (
	windowsBackupBackend       = "windows-task-scheduler"
	windowsBackupSchemaVersion = 1
	windowsBackupPrefix        = "taskscheduler_"
)

var windowsTimeNow = time.Now

type windowsBackupManifest struct {
	Backend       string          `json:"backend"`
	SchemaVersion int             `json:"schema_version"`
	TaskPath      string          `json:"task_path"`
	Jobs          []types.CronJob `json:"jobs"`
}

func createWindowsBackup(cfg config.Config, jobs []types.CronJob) (string, error) {
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	manifest := windowsBackupManifest{
		Backend:       windowsBackupBackend,
		SchemaVersion: windowsBackupSchemaVersion,
		TaskPath:      cfg.WindowsTaskPath,
		Jobs:          jobs,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode Windows backup manifest: %w", err)
	}

	filename := fmt.Sprintf("%s%s.bak", windowsBackupPrefix, windowsTimeNow().Format("20060102_150405"))
	path := filepath.Join(cfg.BackupDir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	if cfg.MaxBackups > 0 {
		if err := pruneWindowsBackups(cfg); err != nil {
			return "", fmt.Errorf("failed to prune backups: %w", err)
		}
	}

	return path, nil
}

func listWindowsBackups(cfg config.Config) ([]types.Backup, error) {
	entries, err := os.ReadDir(cfg.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]types.Backup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), windowsBackupPrefix) || !strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		manifest, err := readWindowsBackupManifest(filepath.Join(cfg.BackupDir, entry.Name()))
		if err != nil {
			continue
		}

		backups = append(backups, types.Backup{
			Filename: entry.Name(),
			Created:  info.ModTime(),
			JobCount: len(manifest.Jobs),
			Size:     info.Size(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Created.After(backups[j].Created)
	})

	return backups, nil
}

func readWindowsBackupManifest(path string) (windowsBackupManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return windowsBackupManifest{}, fmt.Errorf("failed to read backup file: %w", err)
	}

	var manifest windowsBackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return windowsBackupManifest{}, fmt.Errorf("failed to parse Windows backup manifest: %w", err)
	}
	if manifest.Backend != windowsBackupBackend {
		return windowsBackupManifest{}, fmt.Errorf("backup file %s is not a Windows Task Scheduler manifest", filepath.Base(path))
	}
	if manifest.SchemaVersion != windowsBackupSchemaVersion {
		return windowsBackupManifest{}, fmt.Errorf("unsupported Windows backup schema version %d", manifest.SchemaVersion)
	}

	return manifest, nil
}

func pruneWindowsBackups(cfg config.Config) error {
	backups, err := listWindowsBackups(cfg)
	if err != nil {
		return err
	}

	if len(backups) <= cfg.MaxBackups {
		return nil
	}

	for _, backup := range backups[cfg.MaxBackups:] {
		if err := os.Remove(filepath.Join(cfg.BackupDir, backup.Filename)); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", backup.Filename, err)
		}
	}

	return nil
}
