package scheduler

import (
	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

type windowsBackend struct {
	cfg config.Config
}

func (b *windowsBackend) LoadJobs() ([]types.CronJob, error) {
	return nil, errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) SaveJobs(config.Config, []types.CronJob) error {
	return errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) CreateBackup(config.Config) (string, error) {
	return "", errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) ListBackups(config.Config) ([]types.Backup, error) {
	return nil, errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) RestoreBackup(config.Config, string) error {
	return errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) RemoveAll(config.Config) error {
	return errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) RunNow(int) ([]byte, error) {
	return nil, errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) ValidateManagedSchedule(string) error {
	return errBackendNotImplemented("windows scheduler")
}
