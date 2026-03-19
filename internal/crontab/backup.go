package crontab

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

var (
	readCrontabFn  = ReadCrontab
	createBackupFn = CreateBackup
	timeNow        = time.Now
)

// CreateBackup saves the current crontab to a backup file.
func CreateBackup(cfg config.Config) (string, error) {
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	raw, err := readCrontabFn()
	if err != nil {
		return "", fmt.Errorf("failed to read crontab for backup: %w", err)
	}

	timestamp := timeNow().Format("20060102_150405")
	filename := fmt.Sprintf("crontab_%s.bak", timestamp)
	path := filepath.Join(cfg.BackupDir, filename)

	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}
	if cfg.MaxBackups > 0 {
		if err := PruneBackups(cfg); err != nil {
			return "", fmt.Errorf("failed to prune backups: %w", err)
		}
	}

	return path, nil
}

// ListBackups returns all available backups sorted newest first.
func ListBackups(cfg config.Config) ([]types.Backup, error) {
	entries, err := os.ReadDir(cfg.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []types.Backup
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		created := info.ModTime()
		if parsed, ok := parseBackupTimestamp(entry.Name()); ok {
			created = parsed
		}

		// Count jobs by reading content
		path := filepath.Join(cfg.BackupDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		jobs := ParseCrontab(string(content))

		backups = append(backups, types.Backup{
			Filename: entry.Name(),
			Created:  created,
			JobCount: len(jobs),
			Size:     info.Size(),
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].Created.Equal(backups[j].Created) {
			return backups[i].Filename > backups[j].Filename
		}
		return backups[i].Created.After(backups[j].Created)
	})

	return backups, nil
}

// RestoreBackup restores a crontab from a backup file.
// It creates a backup of the current crontab before restoring.
func RestoreBackup(cfg config.Config, filename string) error {
	path := filepath.Join(cfg.BackupDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Backup current crontab before restoring
	if _, err := createBackupFn(cfg); err != nil {
		return fmt.Errorf("failed to backup current crontab before restore: %w", err)
	}

	return writeRawCrontabFn(normalizeCrontab(string(content)))
}

// PruneBackups keeps only the most recent maxBackups files.
func PruneBackups(cfg config.Config) error {
	backups, err := ListBackups(cfg)
	if err != nil {
		return err
	}

	if len(backups) <= cfg.MaxBackups {
		return nil
	}

	// backups are sorted newest first; remove oldest
	for _, backup := range backups[cfg.MaxBackups:] {
		path := filepath.Join(cfg.BackupDir, backup.Filename)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", backup.Filename, err)
		}
	}

	return nil
}

func parseBackupTimestamp(filename string) (time.Time, bool) {
	if !strings.HasPrefix(filename, "crontab_") || !strings.HasSuffix(filename, ".bak") {
		return time.Time{}, false
	}

	stamp := strings.TrimSuffix(strings.TrimPrefix(filename, "crontab_"), ".bak")
	parsed, err := time.Parse("20060102_150405", stamp)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}
