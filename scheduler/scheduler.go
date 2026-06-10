package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"portal/batch/config"
	"portal/batch/job"
)

type Scheduler struct {
	jobs    map[string]job.Job
	entries []config.JobEntry
}

func New(entries []config.JobEntry) *Scheduler {
	return &Scheduler{
		jobs:    make(map[string]job.Job),
		entries: entries,
	}
}

func (s *Scheduler) Register(j job.Job) {
	s.jobs[j.Name()] = j
}

// Validate checks that every entry in the JSON config has a registered job and
// every registered job has an entry in the JSON config.
func (s *Scheduler) Validate() error {
	configIDs := make(map[string]bool, len(s.entries))
	for _, e := range s.entries {
		configIDs[e.ID] = true
		if _, ok := s.jobs[e.ID]; !ok {
			return fmt.Errorf("job %q found in config but not registered in code", e.ID)
		}
	}
	for id := range s.jobs {
		if !configIDs[id] {
			return fmt.Errorf("job %q registered in code but missing from config", id)
		}
	}
	return nil
}

func (s *Scheduler) Run(ctx context.Context, hour int) error {
	for _, e := range s.entries {
		if !e.Active {
			slog.Debug("job inactive, skipping", "id", e.ID)
			continue
		}
		if !slices.Contains(e.Hours, hour) {
			continue
		}
		slog.Info("running job", "id", e.ID, "hour", hour)
		if err := s.jobs[e.ID].Run(ctx); err != nil {
			return fmt.Errorf("job %q failed: %w", e.ID, err)
		}
	}
	return nil
}
