package crontab

import (
	"fmt"
	"strings"

	"github.com/meru143/crontui/pkg/types"
)

type documentEntryKind int

const (
	entryRaw documentEntryKind = iota
	entryJob
)

type documentEntry struct {
	kind documentEntryKind
	raw  string
	job  jobEntry
}

type jobEntry struct {
	job       types.CronJob
	rawLine   string
	rawPrefix []string
	dirty     bool
}

type pendingAnnotations struct {
	rawLines    []string
	description string
	workingDir  string
	mailto      string
}

// Document preserves the order of raw crontab lines while exposing editable jobs.
type Document struct {
	entries []documentEntry
}

// ParseDocument parses a crontab into a lossless document model.
func ParseDocument(raw string) (*Document, error) {
	lines := strings.Split(raw, "\n")
	doc := &Document{entries: make([]documentEntry, 0, len(lines))}
	id := 1
	var pending pendingAnnotations

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			doc.flushPendingAsRaw(&pending)
			doc.entries = append(doc.entries, documentEntry{kind: entryRaw, raw: line})
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "# description:"):
			pending.rawLines = append(pending.rawLines, line)
			pending.description = strings.TrimSpace(strings.TrimPrefix(trimmed, "# description:"))
			continue
		case strings.HasPrefix(trimmed, "# workingdir:"):
			pending.rawLines = append(pending.rawLines, line)
			pending.workingDir = strings.TrimSpace(strings.TrimPrefix(trimmed, "# workingdir:"))
			continue
		case strings.HasPrefix(trimmed, "# mailto:"):
			pending.rawLines = append(pending.rawLines, line)
			pending.mailto = strings.TrimSpace(strings.TrimPrefix(trimmed, "# mailto:"))
			continue
		}

		if job, ok := parseJobLine(trimmed, line, id, pending, true); ok {
			doc.entries = append(doc.entries, documentEntry{kind: entryJob, job: job})
			id++
			pending = pendingAnnotations{}
			continue
		}

		if job, ok := parseJobLine(trimmed, line, id, pending, false); ok {
			doc.entries = append(doc.entries, documentEntry{kind: entryJob, job: job})
			id++
			pending = pendingAnnotations{}
			continue
		}

		doc.flushPendingAsRaw(&pending)
		doc.entries = append(doc.entries, documentEntry{kind: entryRaw, raw: line})
	}

	return doc, nil
}

// Jobs returns the editable cron jobs in document order.
func (d *Document) Jobs() []types.CronJob {
	jobs := make([]types.CronJob, 0)
	for _, entry := range d.entries {
		if entry.kind != entryJob {
			continue
		}
		jobs = append(jobs, entry.job.job)
	}
	return jobs
}

// ReplaceJobs replaces the editable jobs while keeping raw non-job lines untouched.
func (d *Document) ReplaceJobs(jobs []types.CronJob) error {
	newEntries := make([]documentEntry, 0, len(d.entries))
	jobIndex := 0

	for _, entry := range d.entries {
		if entry.kind != entryJob {
			newEntries = append(newEntries, entry)
			continue
		}

		if jobIndex >= len(jobs) {
			continue
		}

		updated := entry
		updated.job.job = jobs[jobIndex]
		if !sameRenderedJob(entry.job.job, jobs[jobIndex]) {
			updated.job.dirty = true
		}
		newEntries = append(newEntries, updated)
		jobIndex++
	}

	for ; jobIndex < len(jobs); jobIndex++ {
		newEntries = append(newEntries, documentEntry{
			kind: entryJob,
			job: jobEntry{
				job:   jobs[jobIndex],
				dirty: true,
			},
		})
	}

	d.entries = newEntries
	return nil
}

// Render converts the document back into crontab text.
func (d *Document) Render() string {
	lines := make([]string, 0, len(d.entries))
	for _, entry := range d.entries {
		switch entry.kind {
		case entryRaw:
			lines = append(lines, entry.raw)
		case entryJob:
			if !entry.job.dirty {
				lines = append(lines, entry.job.rawPrefix...)
				lines = append(lines, entry.job.rawLine)
				continue
			}

			lines = append(lines, renderManagedJob(entry.job.job)...)
		}
	}

	return strings.Join(lines, "\n")
}

func (d *Document) flushPendingAsRaw(pending *pendingAnnotations) {
	for _, line := range pending.rawLines {
		d.entries = append(d.entries, documentEntry{kind: entryRaw, raw: line})
	}
	*pending = pendingAnnotations{}
}

func parseJobLine(trimmed, rawLine string, id int, pending pendingAnnotations, enabled bool) (jobEntry, bool) {
	lineToParse := trimmed
	if !enabled {
		if !strings.HasPrefix(trimmed, "#") {
			return jobEntry{}, false
		}
		lineToParse = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	}

	var schedule, command string
	switch {
	case isCronLine(lineToParse):
		schedule, command = splitCronLine(lineToParse)
	case strings.HasPrefix(lineToParse, "@"):
		schedule, command = splitAtLine(lineToParse)
	default:
		return jobEntry{}, false
	}

	if command == "" {
		return jobEntry{}, false
	}

	return jobEntry{
		job: types.CronJob{
			ID:          id,
			Schedule:    schedule,
			Command:     command,
			Description: pending.description,
			Enabled:     enabled,
			WorkingDir:  pending.workingDir,
			Mailto:      pending.mailto,
		},
		rawLine:   rawLine,
		rawPrefix: append([]string(nil), pending.rawLines...),
	}, true
}

func sameRenderedJob(a, b types.CronJob) bool {
	return a.Schedule == b.Schedule &&
		a.Command == b.Command &&
		a.Description == b.Description &&
		a.Enabled == b.Enabled &&
		a.WorkingDir == b.WorkingDir &&
		a.Mailto == b.Mailto
}

func renderManagedJob(job types.CronJob) []string {
	lines := make([]string, 0, 4)
	if job.Description != "" {
		lines = append(lines, fmt.Sprintf("# description: %s", job.Description))
	}

	line := fmt.Sprintf("%s %s", job.Schedule, job.Command)
	if !job.Enabled {
		line = "#" + line
	}
	lines = append(lines, line)
	return lines
}
