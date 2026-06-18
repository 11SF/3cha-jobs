package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/robfig/cron/v3"

	"portal/batch/config"
)

// fakeJob is a test implementation of the Job interface.
type fakeJob struct {
	name  string
	runs  int
	err   error
}

func (f *fakeJob) Name() string {
	return f.name
}

func (f *fakeJob) Run(ctx context.Context) error {
	f.runs++
	return f.err
}

func TestMatchesNow(t *testing.T) {
	tests := []struct {
		name     string
		cronExpr string
		testTime time.Time
		want     bool
	}{
		{
			name:     "match at 10:30",
			cronExpr: "30 10 * * *",
			testTime: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			want:     true,
		},
		{
			name:     "no match at 10:29",
			cronExpr: "30 10 * * *",
			testTime: time.Date(2024, 1, 15, 10, 29, 45, 0, time.UTC),
			want:     false,
		},
		{
			name:     "no match at 10:31",
			cronExpr: "30 10 * * *",
			testTime: time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC),
			want:     false,
		},
		{
			name:     "match at midnight (0:00)",
			cronExpr: "0 0 * * *",
			testTime: time.Date(2024, 1, 15, 0, 0, 30, 0, time.UTC),
			want:     true,
		},
		{
			name:     "match every 15 minutes",
			cronExpr: "*/15 * * * *",
			testTime: time.Date(2024, 1, 15, 10, 15, 0, 0, time.UTC),
			want:     true,
		},
		{
			name:     "no match at 10:14 with 15-min schedule",
			cronExpr: "*/15 * * * *",
			testTime: time.Date(2024, 1, 15, 10, 14, 59, 0, time.UTC),
			want:     false,
		},
		{
			name:     "match on specific day of week (Monday)",
			cronExpr: "0 9 * * 1",
			testTime: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC), // 2024-01-15 is Monday
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the cron expression
			parsed, err := cron.ParseStandard(tt.cronExpr)
			if err != nil {
				t.Fatalf("failed to parse cron expression %q: %v", tt.cronExpr, err)
			}

			got := matchesNow(parsed, tt.testTime)
			if got != tt.want {
				t.Errorf("matchesNow(%q, %v) = %v, want %v", tt.cronExpr, tt.testTime, got, tt.want)
			}
		})
	}
}

func TestRunMatchesSchedule(t *testing.T) {
	ctx := context.Background()

	job1 := &fakeJob{name: "job1"}
	job2 := &fakeJob{name: "job2"}

	entries := []config.JobEntry{
		{ID: "job1", Schedule: "30 10 * * *", Active: true},
		{ID: "job2", Schedule: "0 0 * * *", Active: true},
	}

	s := New(entries)
	s.Register(job1)
	s.Register(job2)

	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Test at 10:30:00 - only job1 should run
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if err := s.Run(ctx, testTime); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if job1.runs != 1 {
		t.Errorf("job1 should have run once, got %d runs", job1.runs)
	}
	if job2.runs != 0 {
		t.Errorf("job2 should not have run, got %d runs", job2.runs)
	}

	// Reset and test at 00:00:00 - only job2 should run
	job1.runs = 0
	job2.runs = 0

	testTime = time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if err := s.Run(ctx, testTime); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if job1.runs != 0 {
		t.Errorf("job1 should not have run, got %d runs", job1.runs)
	}
	if job2.runs != 1 {
		t.Errorf("job2 should have run once, got %d runs", job2.runs)
	}
}

func TestInactiveJobsNeverRun(t *testing.T) {
	ctx := context.Background()

	job1 := &fakeJob{name: "job1"}
	job2 := &fakeJob{name: "job2"}

	entries := []config.JobEntry{
		{ID: "job1", Schedule: "0 10 * * *", Active: true},
		{ID: "job2", Schedule: "0 10 * * *", Active: false}, // inactive
	}

	s := New(entries)
	s.Register(job1)
	s.Register(job2)

	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// At 10:00:00, both schedules match, but job2 is inactive
	testTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	if err := s.Run(ctx, testTime); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if job1.runs != 1 {
		t.Errorf("job1 should have run once, got %d runs", job1.runs)
	}
	if job2.runs != 0 {
		t.Errorf("job2 (inactive) should not have run, got %d runs", job2.runs)
	}
}

func TestValidateRejectsMalformedCron(t *testing.T) {
	job1 := &fakeJob{name: "job1"}

	entries := []config.JobEntry{
		{ID: "job1", Schedule: "invalid cron syntax", Active: true},
	}

	s := New(entries)
	s.Register(job1)

	err := s.Validate()
	if err == nil {
		t.Errorf("Validate() should reject malformed cron expression, got nil error")
	}
	if err != nil && err.Error() != "invalid cron expression for job \"job1\": invalid expression format" {
		// Check that the error mentions both the job ID and invalid expression
		errStr := err.Error()
		if !contains(errStr, "job1") || !contains(errStr, "invalid") {
			t.Errorf("Validate() error should mention job ID and cron validity, got: %v", err)
		}
	}
}

func TestValidateRejectsMissingJobInCode(t *testing.T) {
	job1 := &fakeJob{name: "job1"}

	entries := []config.JobEntry{
		{ID: "job1", Schedule: "0 10 * * *", Active: true},
		{ID: "job2", Schedule: "0 12 * * *", Active: true}, // not registered
	}

	s := New(entries)
	s.Register(job1)

	err := s.Validate()
	if err == nil {
		t.Errorf("Validate() should reject config entry with no registered job")
	}
	if err != nil && !contains(err.Error(), "job2") {
		t.Errorf("Validate() error should mention job2, got: %v", err)
	}
}

func TestValidateRejectsMissingJobInConfig(t *testing.T) {
	job1 := &fakeJob{name: "job1"}
	job2 := &fakeJob{name: "job2"}

	entries := []config.JobEntry{
		{ID: "job1", Schedule: "0 10 * * *", Active: true},
	}

	s := New(entries)
	s.Register(job1)
	s.Register(job2) // registered but not in config

	err := s.Validate()
	if err == nil {
		t.Errorf("Validate() should reject registered job not in config")
	}
	if err != nil && !contains(err.Error(), "job2") {
		t.Errorf("Validate() error should mention job2, got: %v", err)
	}
}

func TestMultipleJobsInOneRun(t *testing.T) {
	ctx := context.Background()

	job1 := &fakeJob{name: "job1"}
	job2 := &fakeJob{name: "job2"}
	job3 := &fakeJob{name: "job3"}

	entries := []config.JobEntry{
		{ID: "job1", Schedule: "15 10 * * *", Active: true},
		{ID: "job2", Schedule: "15 10 * * *", Active: true}, // same schedule
		{ID: "job3", Schedule: "30 10 * * *", Active: true}, // different schedule
	}

	s := New(entries)
	s.Register(job1)
	s.Register(job2)
	s.Register(job3)

	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// At 10:15:00, job1 and job2 should run, but not job3
	testTime := time.Date(2024, 1, 15, 10, 15, 0, 0, time.UTC)
	if err := s.Run(ctx, testTime); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	if job1.runs != 1 {
		t.Errorf("job1 should have run once, got %d runs", job1.runs)
	}
	if job2.runs != 1 {
		t.Errorf("job2 should have run once, got %d runs", job2.runs)
	}
	if job3.runs != 0 {
		t.Errorf("job3 should not have run, got %d runs", job3.runs)
	}
}

func TestJobFailureStopsScheduler(t *testing.T) {
	ctx := context.Background()

	failingJob := &fakeJob{name: "failing_job", err: errTestFailure}
	job2 := &fakeJob{name: "job2"}

	entries := []config.JobEntry{
		{ID: "failing_job", Schedule: "0 10 * * *", Active: true},
		{ID: "job2", Schedule: "0 10 * * *", Active: true},
	}

	s := New(entries)
	s.Register(failingJob)
	s.Register(job2)

	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	err := s.Run(ctx, testTime)

	if err == nil {
		t.Errorf("Run() should return error when job fails")
	}
	if err != nil && !contains(err.Error(), "failing_job") {
		t.Errorf("Run() error should mention failing job, got: %v", err)
	}

	// job2 should not have run because the scheduler stops on first failure
	if job2.runs != 0 {
		t.Errorf("job2 should not have run after earlier job failure, got %d runs", job2.runs)
	}
}

// Helpers

var errTestFailure = errType("test failure")

type errType string

func (e errType) Error() string { return string(e) }

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findIndex(s, substr) >= 0))
}

func findIndex(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
