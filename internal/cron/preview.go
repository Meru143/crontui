package cron

import (
	"fmt"
	"math"
	"time"
)

// NextRuns returns the next n run times for a cron expression.
func NextRuns(expr string, n int) ([]time.Time, error) {
	if n <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}

	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	times := make([]time.Time, 0, n)
	next := time.Now()
	for i := 0; i < n; i++ {
		next = sched.Next(next)
		if next.IsZero() {
			break
		}
		times = append(times, next)
	}
	return times, nil
}

// HumanReadable returns a human-friendly relative time string.
func HumanReadable(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)

	if diff < 0 {
		return t.Format("Mon Jan 2 15:04")
	}

	seconds := diff.Seconds()
	minutes := diff.Minutes()
	hours := diff.Hours()
	days := hours / 24

	switch {
	case seconds < 60:
		return "in less than a minute"
	case minutes < 60:
		m := int(math.Round(minutes))
		if m == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", m)
	case hours < 24:
		h := int(math.Round(hours))
		if h == 1 {
			return "in 1 hour"
		}
		return fmt.Sprintf("in %d hours", h)
	case days < 7:
		d := int(math.Round(days))
		if d == 1 {
			return "tomorrow at " + t.Format("15:04")
		}
		return fmt.Sprintf("in %d days (%s)", d, t.Format("Mon 15:04"))
	default:
		return t.Format("Mon Jan 2 15:04")
	}
}

// Preview returns a PreviewInfo with validation and next runs.
func Preview(expr string, n int) (valid bool, nextRuns []time.Time, humanReadable []string, errMsg string) {
	runs, err := NextRuns(expr, n)
	if err != nil {
		return false, nil, nil, err.Error()
	}

	hr := make([]string, len(runs))
	for i, t := range runs {
		hr[i] = HumanReadable(t)
	}

	return true, runs, hr, ""
}
