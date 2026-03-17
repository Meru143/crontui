package types

import "time"

// CronJob represents a single cron job entry.
type CronJob struct {
	ID          int
	Schedule    string
	Command     string
	Description string
	Enabled     bool
	WorkingDir  string
	Mailto      string
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
