package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"portal/batch/config"
	"portal/batch/job"
)

type Scheduler struct {
	jobs      map[string]job.Job
	entries   []config.JobEntry
	schedules map[string]cron.Schedule // parsed cron expressions, keyed by entry ID
}

func New(entries []config.JobEntry) *Scheduler {
	return &Scheduler{
		jobs:      make(map[string]job.Job),
		entries:   entries,
		schedules: make(map[string]cron.Schedule),
	}
}

func (s *Scheduler) Register(j job.Job) {
	s.jobs[j.Name()] = j
}

// Validate checks that every entry in the JSON config has a registered job and
// every registered job has an entry in the JSON config. It also parses and
// caches all cron schedules, failing if any schedule syntax is invalid.
func (s *Scheduler) Validate() error {
	configIDs := make(map[string]bool, len(s.entries))
	for _, e := range s.entries {
		configIDs[e.ID] = true
		if _, ok := s.jobs[e.ID]; !ok {
			return fmt.Errorf("job %q found in config but not registered in code", e.ID)
		}
		// Parse and cache the cron schedule for active entries
		if e.Active {
			parsed, err := cron.ParseStandard(e.Schedule)
			if err != nil {
				return fmt.Errorf("invalid cron expression for job %q: %w", e.ID, err)
			}
			s.schedules[e.ID] = parsed
		}
	}
	for id := range s.jobs {
		if !configIDs[id] {
			return fmt.Errorf("job %q registered in code but missing from config", id)
		}
	}
	return nil
}

// matchesNow checks if the given cron schedule matches the current minute.
// It truncates now to the minute boundary and checks if the schedule would
// fire at that exact minute.
func matchesNow(s cron.Schedule, now time.Time) bool {
	truncated := now.Truncate(time.Minute)
	return s.Next(truncated.Add(-time.Second)).Equal(truncated)
}

// Run executes all active jobs whose cron schedule matches the given time.
// Time is truncated to minute-level granularity. Returns an error if any job fails.
func (s *Scheduler) Run(ctx context.Context, now time.Time) error {
	for _, e := range s.entries {
		if !e.Active {
			slog.Debug("job inactive, skipping", "id", e.ID)
			continue
		}
		schedule, ok := s.schedules[e.ID]
		if !ok {
			// This should never happen if Validate() was called, but be defensive
			slog.Debug("schedule not found for job (possibly not active at validation)", "id", e.ID)
			continue
		}
		if !matchesNow(schedule, now) {
			continue
		}
		slog.Info("running job", "id", e.ID, "time", now.Format("2006-01-02 15:04:05"))
		if err := s.jobs[e.ID].Run(ctx); err != nil {
			return fmt.Errorf("job %q failed: %w", e.ID, err)
		}
	}
	return nil
}
