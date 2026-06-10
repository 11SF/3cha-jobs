package access

import (
	"context"

	"github.com/uptrace/bun"
)

type HolidayStorage interface {
	IsHoliday(ctx context.Context, date string) bool
}

type holidayStorage struct {
	db *bun.DB
}

func NewHolidayStorage(db *bun.DB) HolidayStorage {
	return &holidayStorage{db: db}
}

func (s *holidayStorage) IsHoliday(ctx context.Context, date string) bool {
	exists, _ := s.db.NewSelect().Model((*Holiday)(nil)).
		Where("holiday_date::date = ?", date).
		Exists(ctx)
	return exists
}
