package scheduler

import (
	"testing"

	"github.com/meru143/crontui/internal/config"
)

func TestNewBackend_SelectsUnixBackendOnUnixPlatforms(t *testing.T) {
	backend := NewBackend("linux", config.DefaultConfig())

	if _, ok := backend.(*unixBackend); !ok {
		t.Fatalf("backend type = %T, want *unixBackend", backend)
	}
}

func TestNewBackend_SelectsWindowsBackendOnWindows(t *testing.T) {
	backend := NewBackend("windows", config.DefaultConfig())

	if _, ok := backend.(*windowsBackend); !ok {
		t.Fatalf("backend type = %T, want *windowsBackend", backend)
	}
}
