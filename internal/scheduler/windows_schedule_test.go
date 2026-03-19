package scheduler

import (
	"testing"

	"github.com/meru143/crontui/pkg/types"
)

func TestBuildWindowsTaskSpec_Reboot(t *testing.T) {
	spec, err := buildWindowsTaskSpec(`\CronTUI\`, types.CronJob{
		ID:       6,
		Schedule: "@reboot",
		Command:  "Write-Output hello",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("buildWindowsTaskSpec returned error: %v", err)
	}

	if spec.TaskName != "job-6" {
		t.Fatalf("TaskName = %q, want %q", spec.TaskName, "job-6")
	}
	if spec.Source != encodeTaskSource("@reboot") {
		t.Fatalf("Source = %q, want %q", spec.Source, encodeTaskSource("@reboot"))
	}
	if len(spec.Triggers) != 1 {
		t.Fatalf("trigger count = %d, want 1", len(spec.Triggers))
	}
	if spec.Triggers[0].Type != windowsTriggerStartup {
		t.Fatalf("trigger type = %q, want %q", spec.Triggers[0].Type, windowsTriggerStartup)
	}
}

func TestBuildWindowsTaskSpec_DescriptorsAndSupportedCron(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		check    func(t *testing.T, trigger windowsTriggerSpec)
	}{
		{
			name:     "hourly descriptor",
			schedule: "@hourly",
			check: func(t *testing.T, trigger windowsTriggerSpec) {
				if trigger.Type != windowsTriggerDaily || trigger.EveryMinutes != 60 || trigger.AtMinute != 0 {
					t.Fatalf("unexpected trigger: %+v", trigger)
				}
			},
		},
		{
			name:     "minute step",
			schedule: "*/15 * * * *",
			check: func(t *testing.T, trigger windowsTriggerSpec) {
				if trigger.Type != windowsTriggerDaily || trigger.EveryMinutes != 15 || trigger.AtHour != 0 || trigger.AtMinute != 0 {
					t.Fatalf("unexpected trigger: %+v", trigger)
				}
			},
		},
		{
			name:     "daily fixed time",
			schedule: "30 2 * * *",
			check: func(t *testing.T, trigger windowsTriggerSpec) {
				if trigger.Type != windowsTriggerDaily || trigger.AtHour != 2 || trigger.AtMinute != 30 || trigger.EveryMinutes != 0 {
					t.Fatalf("unexpected trigger: %+v", trigger)
				}
			},
		},
		{
			name:     "weekly weekdays",
			schedule: "0 9 * * 1-5",
			check: func(t *testing.T, trigger windowsTriggerSpec) {
				if trigger.Type != windowsTriggerWeekly || trigger.AtHour != 9 || trigger.AtMinute != 0 {
					t.Fatalf("unexpected trigger: %+v", trigger)
				}
				want := []int{1, 2, 3, 4, 5}
				if len(trigger.Weekdays) != len(want) {
					t.Fatalf("Weekdays = %+v, want %+v", trigger.Weekdays, want)
				}
				for i := range want {
					if trigger.Weekdays[i] != want[i] {
						t.Fatalf("Weekdays = %+v, want %+v", trigger.Weekdays, want)
					}
				}
			},
		},
		{
			name:     "monthly first day",
			schedule: "0 0 1 * *",
			check: func(t *testing.T, trigger windowsTriggerSpec) {
				if trigger.Type != windowsTriggerMonthly || trigger.DayOfMonth != 1 || trigger.AtHour != 0 || trigger.AtMinute != 0 {
					t.Fatalf("unexpected trigger: %+v", trigger)
				}
			},
		},
		{
			name:     "yearly january first",
			schedule: "0 0 1 1 *",
			check: func(t *testing.T, trigger windowsTriggerSpec) {
				if trigger.Type != windowsTriggerYearly || trigger.DayOfMonth != 1 || trigger.AtHour != 0 || trigger.AtMinute != 0 {
					t.Fatalf("unexpected trigger: %+v", trigger)
				}
				if len(trigger.Months) != 1 || trigger.Months[0] != 1 {
					t.Fatalf("Months = %+v, want [1]", trigger.Months)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := buildWindowsTaskSpec(`\CronTUI\`, types.CronJob{
				ID:       42,
				Schedule: tt.schedule,
				Command:  "Write-Output hello",
				Enabled:  true,
			})
			if err != nil {
				t.Fatalf("buildWindowsTaskSpec returned error: %v", err)
			}
			if len(spec.Triggers) != 1 {
				t.Fatalf("trigger count = %d, want 1", len(spec.Triggers))
			}
			tt.check(t, spec.Triggers[0])
		})
	}
}

func TestBuildWindowsTaskSpec_RejectsUnsupportedSchedules(t *testing.T) {
	for _, schedule := range []string{
		"0 9 1 * 1",
		"*/7 * * * *",
		"0 9 1-5 * *",
	} {
		t.Run(schedule, func(t *testing.T) {
			_, err := buildWindowsTaskSpec(`\CronTUI\`, types.CronJob{
				ID:       42,
				Schedule: schedule,
				Command:  "Write-Output hello",
				Enabled:  true,
			})
			if err == nil {
				t.Fatalf("buildWindowsTaskSpec(%q) should fail", schedule)
			}
		})
	}
}
