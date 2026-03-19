package cron

import (
	"fmt"
	"strings"

	cronlib "github.com/robfig/cron/v3"
)

// parser supports standard 5-field cron and optional descriptors (@hourly, etc.)
var parser = cronlib.NewParser(
	cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow | cronlib.Descriptor,
)

func normalizeExpr(expr string) string {
	return strings.TrimSpace(expr)
}

func isRebootDescriptor(expr string) bool {
	return strings.EqualFold(normalizeExpr(expr), "@reboot")
}

// Validate checks whether a cron expression is syntactically valid.
func Validate(expr string) (bool, error) {
	expr = normalizeExpr(expr)
	if isRebootDescriptor(expr) {
		return true, nil
	}

	_, err := parser.Parse(expr)
	if err != nil {
		return false, fmt.Errorf("invalid cron expression: %w", err)
	}
	return true, nil
}
