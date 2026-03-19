package scheduler

import (
	"fmt"
	"runtime"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

// Backend abstracts job storage and execution across supported schedulers.
type Backend interface {
	LoadJobs() ([]types.CronJob, error)
	SaveJobs(cfg config.Config, jobs []types.CronJob) error
	CreateBackup(cfg config.Config) (string, error)
	ListBackups(cfg config.Config) ([]types.Backup, error)
	RestoreBackup(cfg config.Config, filename string) error
	RemoveAll(cfg config.Config) error
	RunNow(id int) ([]byte, error)
	ValidateManagedSchedule(expr string) error
}

// NewBackend selects the platform backend for the requested OS.
func NewBackend(goos string, cfg config.Config) Backend {
	if goos == "windows" {
		return &windowsBackend{cfg: cfg}
	}
	return &unixBackend{}
}

// DefaultBackend selects the platform backend for the current runtime.
func DefaultBackend(cfg config.Config) Backend {
	return NewBackend(runtime.GOOS, cfg)
}

func errBackendNotImplemented(name string) error {
	return fmt.Errorf("%s backend is not implemented yet", name)
}
