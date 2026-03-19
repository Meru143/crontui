package scheduler

import (
	"fmt"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/internal/crontab"
	"github.com/meru143/crontui/pkg/types"
)

type unixBackend struct{}

func (b *unixBackend) LoadJobs() ([]types.CronJob, error) {
	raw, err := crontab.ReadCrontab()
	if err != nil {
		return nil, err
	}
	return crontab.ParseCrontab(raw), nil
}

func (b *unixBackend) SaveJobs(cfg config.Config, jobs []types.CronJob) error {
	return crontab.WriteJobsWithBackup(cfg, jobs)
}

func (b *unixBackend) CreateBackup(cfg config.Config) (string, error) {
	return crontab.CreateBackup(cfg)
}

func (b *unixBackend) ListBackups(cfg config.Config) ([]types.Backup, error) {
	return crontab.ListBackups(cfg)
}

func (b *unixBackend) RestoreBackup(cfg config.Config, filename string) error {
	return crontab.RestoreBackup(cfg, filename)
}

func (b *unixBackend) RemoveAll(cfg config.Config) error {
	return crontab.RemoveCrontabWithBackup(cfg)
}

func (b *unixBackend) RunNow(id int) ([]byte, error) {
	raw, err := crontab.ReadCrontab()
	if err != nil {
		return nil, err
	}

	jobs := crontab.ParseCrontab(raw)
	for _, job := range jobs {
		if job.ID != id {
			continue
		}
		if !job.Enabled {
			return nil, fmt.Errorf("job #%d is disabled", id)
		}
		return crontab.ExecCommand(job.Command).CombinedOutput()
	}

	return nil, fmt.Errorf("job #%d not found", id)
}

func (b *unixBackend) ValidateManagedSchedule(string) error {
	return nil
}
