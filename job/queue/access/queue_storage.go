package access

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type QueueStorage interface {
	TodayEntry(ctx context.Context, date string) (*DailyQueue, error)
	ClosePendingBefore(ctx context.Context, date string)
	CreateEntry(ctx context.Context, e DailyQueue) error
	MarkDone(ctx context.Context, date string) error
}

type queueStorage struct {
	db *bun.DB
}

func NewQueueStorage(db *bun.DB) QueueStorage {
	return &queueStorage{db: db}
}

func (s *queueStorage) TodayEntry(ctx context.Context, date string) (*DailyQueue, error) {
	var q DailyQueue
	err := s.db.NewSelect().Model(&q).
		Where("queue_date::date = ?", date).
		OrderExpr("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &q, err
}

func (s *queueStorage) ClosePendingBefore(ctx context.Context, date string) {
	s.db.NewUpdate().Model((*DailyQueue)(nil)).
		Set("status = ?", "done").
		Where("status = ? AND queue_date::date < ?", "pending", date).
		Exec(ctx)
}

func (s *queueStorage) CreateEntry(ctx context.Context, e DailyQueue) error {
	_, err := s.db.NewInsert().Model(&e).Exec(ctx)
	return err
}

func (s *queueStorage) MarkDone(ctx context.Context, date string) error {
	_, err := s.db.NewUpdate().Model((*DailyQueue)(nil)).
		Set("status = ?", "done").
		Where("queue_date::date = ? AND status = ?", date, "pending").
		Exec(ctx)
	return err
}
