package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"portal/batch/job/queue/access"
)

func (j *QueueJob) Name() string { return "daily_queue" }

func (j *QueueJob) Run(ctx context.Context) error {
	today := time.Now().In(j.loc).Truncate(24 * time.Hour)
	date := today.Format("2006-01-02")

	existing, err := j.queueStorage.TodayEntry(ctx, date)
	if err != nil {
		return err
	}
	if existing != nil {
		slog.Info("entry already exists, skipping", "date", date, "member", existing.MemberID)
		return nil
	}

	if !j.isWorkingDay(ctx, today, date) {
		slog.Info("not a working day, skipping", "date", date)
		return nil
	}

	members, err := j.memberStorage.ActiveMembers(ctx)
	if err != nil {
		return err
	}
	if len(members) == 0 {
		slog.Warn("no active members, skipping")
		return nil
	}

	j.queueStorage.ClosePendingBefore(ctx, date)

	next := roundRobinNext(members, j.memberStorage.LastMemberID(ctx))

	if err := j.queueStorage.CreateEntry(ctx, access.DailyQueue{
		ID:        uuid.New(),
		QueueDate: today,
		MemberID:  next.ID,
		Status:    "pending",
	}); err != nil {
		return err
	}

	slog.Info("created queue entry", "date", date, "member", next.Name)
	return nil
}

func (j *QueueJob) isWorkingDay(ctx context.Context, date time.Time, dateStr string) bool {
	wd := date.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !j.holidayStorage.IsHoliday(ctx, dateStr)
}

func roundRobinNext(members []access.Member, lastID string) access.Member {
	if lastID == "" || len(members) == 1 {
		return members[0]
	}
	for i, m := range members {
		if m.ID.String() == lastID {
			return members[(i+1)%len(members)]
		}
	}
	return members[0]
}
