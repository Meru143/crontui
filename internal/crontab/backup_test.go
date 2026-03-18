package crontab

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/meru143/crontui/internal/config"
)

func TestListBackups_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	backups, err := ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestListBackups_NonExistentDir(t *testing.T) {
	cfg := config.Config{BackupDir: filepath.Join(t.TempDir(), "nope"), MaxBackups: 10}

	backups, err := ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups should not error on missing dir: %v", err)
	}
	if backups != nil {
		t.Errorf("expected nil, got %v", backups)
	}
}

func TestListBackups_SortedNewestFirst(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	// Create 3 backup files with simple crontab content
	names := []string{"crontab_20250101_000000.bak", "crontab_20250301_000000.bak", "crontab_20250201_000000.bak"}
	for _, name := range names {
		content := "0 * * * * /usr/bin/cmd\n"
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	backups, err := ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 3 {
		t.Fatalf("expected 3 backups, got %d", len(backups))
	}

	// Each backup should have 1 job parsed from the content
	for i, b := range backups {
		if b.JobCount != 1 {
			t.Errorf("backup[%d].JobCount = %d, want 1", i, b.JobCount)
		}
	}
}

func TestListBackups_IgnoresNonBakFiles(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	os.WriteFile(filepath.Join(tmp, "crontab_20250101_000000.bak"), []byte("0 * * * * cmd\n"), 0o644)
	os.WriteFile(filepath.Join(tmp, "notes.txt"), []byte("not a backup"), 0o644)
	os.MkdirAll(filepath.Join(tmp, "subdir"), 0o755)

	backups, err := ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(backups))
	}
}

func TestPruneBackups(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 2}

	// Create 4 backup files
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("crontab_2025010%d_000000.bak", i)
		os.WriteFile(filepath.Join(tmp, name), []byte("0 * * * * /cmd\n"), 0o644)
	}

	err := PruneBackups(cfg)
	if err != nil {
		t.Fatalf("PruneBackups: %v", err)
	}

	remaining, _ := ListBackups(cfg)
	if len(remaining) != 2 {
		t.Errorf("expected 2 backups after prune, got %d", len(remaining))
	}
}

func TestPruneBackups_NoopWhenUnderLimit(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	os.WriteFile(filepath.Join(tmp, "crontab_20250101_000000.bak"), []byte("0 * * * * cmd\n"), 0o644)

	err := PruneBackups(cfg)
	if err != nil {
		t.Fatalf("PruneBackups: %v", err)
	}

	remaining, _ := ListBackups(cfg)
	if len(remaining) != 1 {
		t.Errorf("expected 1 backup (noop prune), got %d", len(remaining))
	}
}
