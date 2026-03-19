package model

import (
	"strings"
	"testing"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg)

	if m.currentView != ViewList {
		t.Errorf("initial view = %d, want ViewList (%d)", m.currentView, ViewList)
	}
	if m.filter != "all" {
		t.Errorf("initial filter = %q, want %q", m.filter, "all")
	}
}

func TestFilteredJobs_All(t *testing.T) {
	m := Model{
		filter: "all",
		jobs: []types.CronJob{
			{ID: 1, Command: "a", Enabled: true},
			{ID: 2, Command: "b", Enabled: false},
		},
	}
	got := m.filteredJobs()
	if len(got) != 2 {
		t.Errorf("filter=all: got %d, want 2", len(got))
	}
}

func TestFilteredJobs_Enabled(t *testing.T) {
	m := Model{
		filter: "enabled",
		jobs: []types.CronJob{
			{ID: 1, Command: "a", Enabled: true},
			{ID: 2, Command: "b", Enabled: false},
			{ID: 3, Command: "c", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 2 {
		t.Errorf("filter=enabled: got %d, want 2", len(got))
	}
}

func TestFilteredJobs_Disabled(t *testing.T) {
	m := Model{
		filter: "disabled",
		jobs: []types.CronJob{
			{ID: 1, Command: "a", Enabled: true},
			{ID: 2, Command: "b", Enabled: false},
		},
	}
	got := m.filteredJobs()
	if len(got) != 1 {
		t.Errorf("filter=disabled: got %d, want 1", len(got))
	}
	if got[0].ID != 2 {
		t.Errorf("disabled job ID = %d, want 2", got[0].ID)
	}
}

func TestFilteredJobs_Search(t *testing.T) {
	m := Model{
		filter:      "all",
		searchQuery: "backup",
		jobs: []types.CronJob{
			{ID: 1, Command: "/usr/bin/backup", Enabled: true},
			{ID: 2, Command: "/usr/bin/cleanup", Enabled: true},
			{ID: 3, Command: "/usr/bin/other", Description: "Backup helper", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 2 {
		t.Errorf("search=backup: got %d, want 2", len(got))
	}
}

func TestFilteredJobs_SearchCaseInsensitive(t *testing.T) {
	m := Model{
		filter:      "all",
		searchQuery: "BACKUP",
		jobs: []types.CronJob{
			{ID: 1, Command: "/usr/bin/backup", Enabled: true},
			{ID: 2, Command: "/usr/bin/cleanup", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 1 {
		t.Errorf("case-insensitive search: got %d, want 1", len(got))
	}
}

func TestFilteredJobs_SearchAndFilter(t *testing.T) {
	m := Model{
		filter:      "enabled",
		searchQuery: "backup",
		jobs: []types.CronJob{
			{ID: 1, Command: "/usr/bin/backup", Enabled: true},
			{ID: 2, Command: "/usr/bin/backup-old", Enabled: false},
			{ID: 3, Command: "/usr/bin/cleanup", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 1 {
		t.Errorf("search+filter: got %d, want 1", len(got))
	}
}

func TestDemoJobs(t *testing.T) {
	jobs := demoJobs()
	if len(jobs) == 0 {
		t.Fatal("demoJobs() should return non-empty slice")
	}
	for i, j := range jobs {
		if j.ID == 0 {
			t.Errorf("demoJobs[%d].ID should not be 0", i)
		}
		if j.Schedule == "" {
			t.Errorf("demoJobs[%d].Schedule should not be empty", i)
		}
		if j.Command == "" {
			t.Errorf("demoJobs[%d].Command should not be empty", i)
		}
	}
}

func TestViewForm_DoesNotShowWorkingDirField(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd

	view := m.viewForm()
	if strings.Contains(view, "Working Dir") {
		t.Fatalf("viewForm should not show working directory field:\n%s", view)
	}
}

func TestViewForm_DoesNotShowMailtoField(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd

	view := m.viewForm()
	if strings.Contains(view, "Mailto") {
		t.Fatalf("viewForm should not show mailto field:\n%s", view)
	}
}
