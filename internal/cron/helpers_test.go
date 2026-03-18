package cron

import "time"

// fixedPastTime returns a time in the past for testing HumanReadable.
func fixedPastTime() time.Time {
	return time.Now().Add(-24 * time.Hour)
}
