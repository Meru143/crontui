package crontab

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/meru143/crontui/pkg/types"
)

type documentEntryKind int

const (
	entryRaw documentEntryKind = iota
	entryJob
)

const managedIDPrefix = "# crontui:id:"

type documentEntry struct {
	kind documentEntryKind
	raw  string
	job  jobEntry
}

type jobEntry struct {
	job                types.CronJob
	rawLine            string
	rawPrefix          []string
	managedIDPresent   bool
	managedIDCanonical bool
	dirty              bool
}

type pendingAnnotations struct {
	rawLines         []string
	description      string
	workingDir       string
	mailto           string
	managedID        int
	managedIDPresent bool
}

// Document preserves the order of raw crontab lines while exposing editable jobs.
type Document struct {
	entries []documentEntry
}

// ParseDocument parses a crontab into a lossless document model.
func ParseDocument(raw string) (*Document, error) {
	if raw == "" {
		return &Document{}, nil
	}

	lines := strings.Split(raw, "\n")
	doc := &Document{entries: make([]documentEntry, 0, len(lines))}
	reservedIDs := collectReservedManagedIDs(lines)
	usedIDs := make(map[int]struct{}, len(reservedIDs))
	nextFallbackID := 1
	var pending pendingAnnotations

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			doc.flushPendingAsRaw(&pending)
			doc.entries = append(doc.entries, documentEntry{kind: entryRaw, raw: line})
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, managedIDPrefix):
			pending.rawLines = append(pending.rawLines, line)
			pending.managedID, pending.managedIDPresent = parseManagedID(trimmed)
			continue
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

		if job, ok := parseJobLine(trimmed, line, 0, pending, true); ok {
			job.job.ID, job.managedIDPresent, job.managedIDCanonical = selectDocumentJobID(pending, reservedIDs, usedIDs, &nextFallbackID)
			doc.entries = append(doc.entries, documentEntry{kind: entryJob, job: job})
			pending = pendingAnnotations{}
			continue
		}

		if job, ok := parseJobLine(trimmed, line, 0, pending, false); ok {
			job.job.ID, job.managedIDPresent, job.managedIDCanonical = selectDocumentJobID(pending, reservedIDs, usedIDs, &nextFallbackID)
			doc.entries = append(doc.entries, documentEntry{kind: entryJob, job: job})
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
	jobs = normalizeManagedJobIDs(jobs)
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
		if !sameRenderedJob(entry.job.job, jobs[jobIndex]) ||
			!entry.job.managedIDPresent ||
			!entry.job.managedIDCanonical ||
			entry.job.job.ID != jobs[jobIndex].ID {
			updated.job.dirty = true
		}
		updated.job.managedIDPresent = true
		updated.job.managedIDCanonical = true
		newEntries = append(newEntries, updated)
		jobIndex++
	}

	for ; jobIndex < len(jobs); jobIndex++ {
		newEntries = append(newEntries, documentEntry{
			kind: entryJob,
			job: jobEntry{
				job:                jobs[jobIndex],
				managedIDPresent:   true,
				managedIDCanonical: true,
				dirty:              true,
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
		rawLine:            rawLine,
		rawPrefix:          append([]string(nil), pending.rawLines...),
		managedIDPresent:   pending.managedIDPresent,
		managedIDCanonical: pending.managedIDPresent && pending.managedID == id && pending.managedID > 0,
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
	lines := make([]string, 0, 5)
	lines = append(lines, fmt.Sprintf("%s %d", managedIDPrefix, job.ID))
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

func parseManagedID(line string) (int, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(line, managedIDPrefix))
	if value == "" {
		return 0, false
	}

	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func collectReservedManagedIDs(lines []string) map[int]struct{} {
	reserved := make(map[int]struct{})
	for _, line := range lines {
		id, ok := parseManagedID(strings.TrimSpace(line))
		if ok {
			reserved[id] = struct{}{}
		}
	}
	return reserved
}

func nextAvailableManagedID(reservedIDs, usedIDs map[int]struct{}, nextID *int) int {
	for {
		id := *nextID
		*nextID = *nextID + 1
		if _, reserved := reservedIDs[id]; reserved {
			continue
		}
		if _, used := usedIDs[id]; used {
			continue
		}
		return id
	}
}

func selectDocumentJobID(pending pendingAnnotations, reservedIDs, usedIDs map[int]struct{}, nextFallbackID *int) (id int, present bool, canonical bool) {
	if pending.managedIDPresent && pending.managedID > 0 {
		if _, exists := usedIDs[pending.managedID]; !exists {
			usedIDs[pending.managedID] = struct{}{}
			return pending.managedID, true, true
		}
	}

	id = nextAvailableManagedID(reservedIDs, usedIDs, nextFallbackID)
	usedIDs[id] = struct{}{}
	return id, pending.managedIDPresent, false
}

func normalizeManagedJobIDs(jobs []types.CronJob) []types.CronJob {
	normalized := append([]types.CronJob(nil), jobs...)
	used := make(map[int]struct{}, len(normalized))
	nextID := 1

	for i := range normalized {
		id := normalized[i].ID
		if id > 0 {
			if _, exists := used[id]; !exists {
				used[id] = struct{}{}
				if id >= nextID {
					nextID = id + 1
				}
				continue
			}
		}

		for {
			if _, exists := used[nextID]; !exists {
				break
			}
			nextID++
		}
		normalized[i].ID = nextID
		used[nextID] = struct{}{}
		nextID++
	}

	return normalized
}
