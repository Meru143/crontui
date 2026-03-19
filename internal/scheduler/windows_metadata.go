package scheduler

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	taskNamePrefix   = "job-"
	taskSourcePrefix = "crontui:v1;schedule="
)

func taskNameForID(id int) string {
	return fmt.Sprintf("%s%d", taskNamePrefix, id)
}

func parseIDFromTaskName(name string) (int, error) {
	if !strings.HasPrefix(name, taskNamePrefix) {
		return 0, fmt.Errorf("task name %q is not a managed CronTUI task", name)
	}

	idPart := strings.TrimPrefix(name, taskNamePrefix)
	if idPart == "" {
		return 0, fmt.Errorf("task name %q is missing an ID", name)
	}

	id, err := strconv.Atoi(idPart)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("task name %q has an invalid ID", name)
	}

	return id, nil
}

func encodeTaskSource(schedule string) string {
	return taskSourcePrefix + base64.StdEncoding.EncodeToString([]byte(schedule))
}

func decodeTaskSource(source string) (string, error) {
	if !strings.HasPrefix(source, taskSourcePrefix) {
		return "", fmt.Errorf("task source %q is not managed CronTUI metadata", source)
	}

	encoded := strings.TrimPrefix(source, taskSourcePrefix)
	if encoded == "" {
		return "", fmt.Errorf("task source %q is missing the encoded schedule", source)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode task source schedule: %w", err)
	}

	schedule := string(decoded)
	if schedule == "" {
		return "", fmt.Errorf("task source %q decodes to an empty schedule", source)
	}

	return schedule, nil
}
