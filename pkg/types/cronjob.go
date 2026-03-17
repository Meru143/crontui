package types

import "time"

// CronJob represents a single cron job entry.
type CronJob struct {
	ID          int
	Schedule    string
	Command     string
	Description string
	Enabled     bool
	EnvVars     map[string]string
	WorkingDir  string
	Mailto      string
}

// FormData holds form field values during add/edit.
type FormData struct {
	Schedule    string
	Command     string
	Description string
	WorkingDir  string
	Mailto      string
	ActiveField int // 0=schedule, 1=command, 2=description, 3=workingDir, 4=mailto
}

// Backup represents a saved crontab backup file.
type Backup struct {
	Filename string
	Created  time.Time
	JobCount int
	Size     int64
}

// PreviewInfo holds cron expression validation and preview data.
type PreviewInfo struct {
	Schedule      string
	NextRuns      []time.Time
	HumanReadable []string
	Valid         bool
	Error         string
}
