package cron

import (
	"fmt"

	cronlib "github.com/robfig/cron/v3"
)

// parser supports standard 5-field cron and optional descriptors (@hourly, etc.)
var parser = cronlib.NewParser(
	cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow | cronlib.Descriptor,
)

// Validate checks whether a cron expression is syntactically valid.
func Validate(expr string) (bool, error) {
	_, err := parser.Parse(expr)
	if err != nil {
		return false, fmt.Errorf("invalid cron expression: %w", err)
	}
	return true, nil
}
