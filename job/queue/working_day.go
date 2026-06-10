package queue

import (
	"context"
	"time"

	"portal/batch/job/queue/access"
)

func isWorkingDay(ctx context.Context, hs access.HolidayStorage, date time.Time, dateStr string) bool {
	wd := date.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return !hs.IsHoliday(ctx, dateStr)
}
