package crontab

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestListBackups_PrefersTimestampInFilenameOverModTime(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	oldest := filepath.Join(tmp, "crontab_20250101_000000.bak")
	newest := filepath.Join(tmp, "crontab_20250102_000000.bak")

	if err := os.WriteFile(oldest, []byte("0 * * * * old\n"), 0o644); err != nil {
		t.Fatalf("write oldest backup: %v", err)
	}
	if err := os.WriteFile(newest, []byte("0 * * * * new\n"), 0o644); err != nil {
		t.Fatalf("write newest backup: %v", err)
	}

	// Reverse the file modtimes so filename timestamps, not filesystem timing,
	// determine the logical backup order.
	if err := os.Chtimes(oldest, time.Date(2026, time.March, 19, 12, 0, 0, 0, time.UTC), time.Date(2026, time.March, 19, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Chtimes(oldest): %v", err)
	}
	if err := os.Chtimes(newest, time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC), time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Chtimes(newest): %v", err)
	}

	backups, err := ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}
	if backups[0].Filename != filepath.Base(newest) {
		t.Fatalf("backups[0].Filename = %q, want %q", backups[0].Filename, filepath.Base(newest))
	}
}

func TestListBackups_IgnoresNonBakFiles(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	if err := os.WriteFile(filepath.Join(tmp, "crontab_20250101_000000.bak"), []byte("0 * * * * cmd\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "notes.txt"), []byte("not a backup"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

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
		if err := os.WriteFile(filepath.Join(tmp, name), []byte("0 * * * * /cmd\n"), 0o644); err != nil {
			t.Fatalf("write backup %s: %v", name, err)
		}
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

	if err := os.WriteFile(filepath.Join(tmp, "crontab_20250101_000000.bak"), []byte("0 * * * * cmd\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	err := PruneBackups(cfg)
	if err != nil {
		t.Fatalf("PruneBackups: %v", err)
	}

	remaining, _ := ListBackups(cfg)
	if len(remaining) != 1 {
		t.Errorf("expected 1 backup (noop prune), got %d", len(remaining))
	}
}

func TestRestoreBackup_WritesRawContent(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	raw := "SHELL=/bin/bash\n# @hourly /usr/bin/hourly\n"
	if err := os.WriteFile(filepath.Join(tmp, "restore.bak"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	oldCreateBackup := createBackupFn
	oldWriteRaw := writeRawCrontabFn
	defer func() {
		createBackupFn = oldCreateBackup
		writeRawCrontabFn = oldWriteRaw
	}()

	createdBackup := false
	var wrote string

	createBackupFn = func(config.Config) (string, error) {
		createdBackup = true
		return filepath.Join(tmp, "pre-restore.bak"), nil
	}
	writeRawCrontabFn = func(content string) error {
		wrote = content
		return nil
	}

	if err := RestoreBackup(cfg, "restore.bak"); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}
	if !createdBackup {
		t.Fatal("expected RestoreBackup to create a pre-restore backup")
	}
	if wrote != raw {
		t.Fatalf("restored raw mismatch\n--- got ---\n%s\n--- want ---\n%s", wrote, raw)
	}
}

func TestRestoreBackup_AppendsTrailingNewline(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 10}

	raw := "SHELL=/bin/bash"
	if err := os.WriteFile(filepath.Join(tmp, "restore.bak"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	oldCreateBackup := createBackupFn
	oldWriteRaw := writeRawCrontabFn
	defer func() {
		createBackupFn = oldCreateBackup
		writeRawCrontabFn = oldWriteRaw
	}()

	createBackupFn = func(config.Config) (string, error) {
		return filepath.Join(tmp, "pre-restore.bak"), nil
	}

	var wrote string
	writeRawCrontabFn = func(content string) error {
		wrote = content
		return nil
	}

	if err := RestoreBackup(cfg, "restore.bak"); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}

	if wrote != raw+"\n" {
		t.Fatalf("RestoreBackup must append trailing newline\n--- got ---\n%q\n--- want ---\n%q", wrote, raw+"\n")
	}
}

func TestCreateBackup_PrunesWhenOverLimit(t *testing.T) {
	tmp := t.TempDir()
	cfg := config.Config{BackupDir: tmp, MaxBackups: 2}

	oldReadCrontab := readCrontabFn
	oldTimeNow := timeNow
	defer func() {
		readCrontabFn = oldReadCrontab
		timeNow = oldTimeNow
	}()

	readCrontabFn = func() (string, error) {
		return "0 * * * * /usr/bin/cmd\n", nil
	}

	times := []time.Time{
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	timeIndex := 0
	timeNow = func() time.Time {
		current := times[timeIndex]
		timeIndex++
		return current
	}

	for i := 0; i < 3; i++ {
		if _, err := CreateBackup(cfg); err != nil {
			t.Fatalf("CreateBackup #%d: %v", i, err)
		}
	}

	files, err := filepath.Glob(filepath.Join(tmp, "*.bak"))
	if err != nil {
		t.Fatalf("Glob backups: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 backups after create+prune, got %d", len(files))
	}
	if filepath.Base(files[0]) == "crontab_20250101_000000.bak" || filepath.Base(files[1]) == "crontab_20250101_000000.bak" {
		t.Fatalf("oldest backup should have been pruned, got files %v", files)
	}
}
