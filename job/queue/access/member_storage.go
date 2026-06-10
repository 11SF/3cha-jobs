package access

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type MemberStorage interface {
	ActiveMembers(ctx context.Context) ([]Member, error)
	LastMemberID(ctx context.Context) string
}

type memberStorage struct {
	db *bun.DB
}

func NewMemberStorage(db *bun.DB) MemberStorage {
	return &memberStorage{db: db}
}

func (s *memberStorage) ActiveMembers(ctx context.Context) ([]Member, error) {
	var members []Member
	err := s.db.NewSelect().Model(&members).
		Where("is_active = true AND deleted_at IS NULL").
		OrderExpr("sort_order ASC, created_at ASC").
		Scan(ctx)
	return members, err
}

func (s *memberStorage) LastMemberID(ctx context.Context) string {
	var cfg QueueConfig
	s.db.NewSelect().Model(&cfg).Where("id = ?", "singleton").Scan(ctx)

	var lastQ DailyQueue
	err := s.db.NewSelect().Model(&lastQ).
		OrderExpr("queue_date DESC, created_at DESC").
		Limit(1).
		Scan(ctx)
	if err == sql.ErrNoRows {
		if cfg.LastMemberID != nil {
			return cfg.LastMemberID.String()
		}
		return ""
	}

	if cfg.LastMemberID != nil && lastQ.CreatedAt.Before(cfg.UpdatedAt) {
		return cfg.LastMemberID.String()
	}
	return lastQ.MemberID.String()
}
