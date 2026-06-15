package queue

import (
	"context"
	"log/slog"
	"time"

	"portal/batch/job/queue/access"
)

type MarkQueueDoneJob struct {
	loc            *time.Location
	queueStorage   access.QueueStorage
	holidayStorage access.HolidayStorage
}

func NewMarkQueueDoneJob(loc *time.Location, qs access.QueueStorage, hs access.HolidayStorage) *MarkQueueDoneJob {
	return &MarkQueueDoneJob{
		loc:            loc,
		queueStorage:   qs,
		holidayStorage: hs,
	}
}

func (j *MarkQueueDoneJob) Name() string { return "mark_queue_done" }

func (j *MarkQueueDoneJob) Run(ctx context.Context) error {
	now := time.Now().In(j.loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, j.loc)
	date := today.Format("2006-01-02")

	if !isWorkingDay(ctx, j.holidayStorage, today, date) {
		slog.Info("not a working day, skipping", "job", j.Name(), "date", date)
		return nil
	}

	entry, err := j.queueStorage.TodayEntry(ctx, date)
	if err != nil {
		return err
	}
	if entry == nil {
		slog.Info("no queue entry for today", "date", date)
		return nil
	}
	if entry.Status != "pending" {
		slog.Info("queue entry already done", "date", date)
		return nil
	}

	if err := j.queueStorage.MarkDone(ctx, date); err != nil {
		return err
	}

	slog.Info("marked queue entry as done", "date", date)
	return nil
}
