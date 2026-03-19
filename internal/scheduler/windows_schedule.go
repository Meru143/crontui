package scheduler

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/meru143/crontui/pkg/types"
)

type windowsTriggerType string

const (
	windowsTriggerStartup windowsTriggerType = "startup"
	windowsTriggerDaily   windowsTriggerType = "daily"
	windowsTriggerWeekly  windowsTriggerType = "weekly"
	windowsTriggerMonthly windowsTriggerType = "monthly"
	windowsTriggerYearly  windowsTriggerType = "yearly"
)

type windowsTaskSpec struct {
	TaskPath    string
	TaskName    string
	Description string
	Source      string
	Triggers    []windowsTriggerSpec
	Enabled     bool
	Command     string
}

type windowsTriggerSpec struct {
	Type         windowsTriggerType
	AtHour       int
	AtMinute     int
	EveryMinutes int
	Weekdays     []int
	DayOfMonth   int
	Months       []int
}

func buildWindowsTaskSpec(taskPath string, job types.CronJob) (windowsTaskSpec, error) {
	trigger, err := translateWindowsSchedule(job.Schedule)
	if err != nil {
		return windowsTaskSpec{}, err
	}

	return windowsTaskSpec{
		TaskPath:    taskPath,
		TaskName:    taskNameForID(job.ID),
		Description: job.Description,
		Source:      encodeTaskSource(job.Schedule),
		Triggers:    []windowsTriggerSpec{trigger},
		Enabled:     job.Enabled,
		Command:     job.Command,
	}, nil
}

func translateWindowsSchedule(schedule string) (windowsTriggerSpec, error) {
	switch schedule {
	case "@reboot":
		return windowsTriggerSpec{Type: windowsTriggerStartup}, nil
	case "@hourly":
		return windowsTriggerSpec{Type: windowsTriggerDaily, AtHour: 0, AtMinute: 0, EveryMinutes: 60}, nil
	case "@daily":
		return windowsTriggerSpec{Type: windowsTriggerDaily, AtHour: 0, AtMinute: 0}, nil
	case "@weekly":
		return windowsTriggerSpec{Type: windowsTriggerWeekly, AtHour: 0, AtMinute: 0, Weekdays: []int{0}}, nil
	case "@monthly":
		return windowsTriggerSpec{Type: windowsTriggerMonthly, AtHour: 0, AtMinute: 0, DayOfMonth: 1}, nil
	case "@yearly", "@annually":
		return windowsTriggerSpec{Type: windowsTriggerYearly, AtHour: 0, AtMinute: 0, DayOfMonth: 1, Months: []int{1}}, nil
	}

	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return windowsTriggerSpec{}, fmt.Errorf("schedule is valid cron but not supported by Windows Task Scheduler backend")
	}

	minuteField, hourField, dayOfMonthField, monthField, dayOfWeekField := fields[0], fields[1], fields[2], fields[3], fields[4]

	if dayOfMonthField != "*" && dayOfWeekField != "*" {
		return windowsTriggerSpec{}, fmt.Errorf("windows backend cannot represent schedules that mix day-of-month and day-of-week")
	}

	minute, err := parseNumericField(minuteField, 0, 59, "minute")
	if err == nil && hourField == "*" && dayOfMonthField == "*" && monthField == "*" && dayOfWeekField == "*" {
		return windowsTriggerSpec{Type: windowsTriggerDaily, AtHour: 0, AtMinute: minute, EveryMinutes: 60}, nil
	}

	if hourField == "*" && dayOfMonthField == "*" && monthField == "*" && dayOfWeekField == "*" {
		step, stepErr := parseMinuteStep(minuteField)
		if stepErr == nil {
			return windowsTriggerSpec{Type: windowsTriggerDaily, AtHour: 0, AtMinute: 0, EveryMinutes: step}, nil
		}
	}

	minute, err = parseNumericField(minuteField, 0, 59, "minute")
	if err != nil {
		return windowsTriggerSpec{}, unsupportedScheduleError()
	}
	hour, err := parseNumericField(hourField, 0, 23, "hour")
	if err != nil {
		return windowsTriggerSpec{}, unsupportedScheduleError()
	}

	switch {
	case dayOfMonthField == "*" && monthField == "*" && dayOfWeekField == "*":
		return windowsTriggerSpec{Type: windowsTriggerDaily, AtHour: hour, AtMinute: minute}, nil

	case dayOfMonthField == "*" && monthField == "*" && dayOfWeekField != "*":
		weekdays, err := parseWeekdays(dayOfWeekField)
		if err != nil {
			return windowsTriggerSpec{}, unsupportedScheduleError()
		}
		return windowsTriggerSpec{Type: windowsTriggerWeekly, AtHour: hour, AtMinute: minute, Weekdays: weekdays}, nil

	case dayOfMonthField != "*" && monthField == "*" && dayOfWeekField == "*":
		dayOfMonth, err := parseNumericField(dayOfMonthField, 1, 31, "day-of-month")
		if err != nil {
			return windowsTriggerSpec{}, unsupportedScheduleError()
		}
		return windowsTriggerSpec{Type: windowsTriggerMonthly, AtHour: hour, AtMinute: minute, DayOfMonth: dayOfMonth}, nil

	case dayOfMonthField != "*" && monthField != "*" && dayOfWeekField == "*":
		dayOfMonth, err := parseNumericField(dayOfMonthField, 1, 31, "day-of-month")
		if err != nil {
			return windowsTriggerSpec{}, unsupportedScheduleError()
		}
		month, err := parseNumericField(monthField, 1, 12, "month")
		if err != nil {
			return windowsTriggerSpec{}, unsupportedScheduleError()
		}
		return windowsTriggerSpec{Type: windowsTriggerYearly, AtHour: hour, AtMinute: minute, DayOfMonth: dayOfMonth, Months: []int{month}}, nil

	default:
		return windowsTriggerSpec{}, unsupportedScheduleError()
	}
}

func parseMinuteStep(field string) (int, error) {
	if !strings.HasPrefix(field, "*/") {
		return 0, fmt.Errorf("minute field %q is not a step schedule", field)
	}

	step, err := strconv.Atoi(strings.TrimPrefix(field, "*/"))
	if err != nil || step <= 0 || step >= 60 {
		return 0, fmt.Errorf("minute step %q is invalid", field)
	}
	if 60%step != 0 {
		return 0, fmt.Errorf("minute step %q cannot be represented exactly", field)
	}
	return step, nil
}

func parseNumericField(field string, minValue, maxValue int, name string) (int, error) {
	value, err := strconv.Atoi(field)
	if err != nil {
		return 0, fmt.Errorf("%s field %q is not a literal value", name, field)
	}
	if value < minValue || value > maxValue {
		return 0, fmt.Errorf("%s field %q is out of range", name, field)
	}
	return value, nil
}

func parseWeekdays(field string) ([]int, error) {
	parts := strings.Split(field, ",")
	values := make([]int, 0, len(parts))
	seen := map[int]bool{}

	for _, part := range parts {
		if strings.Contains(part, "-") {
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("weekday field %q has an invalid range", field)
			}

			start, err := normalizeWeekday(bounds[0])
			if err != nil {
				return nil, err
			}
			end, err := normalizeWeekday(bounds[1])
			if err != nil {
				return nil, err
			}
			if start > end {
				return nil, fmt.Errorf("weekday field %q has a descending range", field)
			}

			for day := start; day <= end; day++ {
				if !seen[day] {
					values = append(values, day)
					seen[day] = true
				}
			}
			continue
		}

		day, err := normalizeWeekday(part)
		if err != nil {
			return nil, err
		}
		if !seen[day] {
			values = append(values, day)
			seen[day] = true
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("weekday field %q is empty", field)
	}

	slices.Sort(values)
	return values, nil
}

func normalizeWeekday(value string) (int, error) {
	day, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("weekday %q is not numeric", value)
	}
	switch day {
	case 7:
		return 0, nil
	case 0, 1, 2, 3, 4, 5, 6:
		return day, nil
	default:
		return 0, fmt.Errorf("weekday %q is out of range", value)
	}
}

func unsupportedScheduleError() error {
	return fmt.Errorf("schedule is valid cron but not supported by Windows Task Scheduler backend")
}
