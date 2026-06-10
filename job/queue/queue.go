package queue

import (
	"time"

	"portal/batch/job/queue/access"
)

type QueueJob struct {
	loc            *time.Location
	queueStorage   access.QueueStorage
	memberStorage  access.MemberStorage
	holidayStorage access.HolidayStorage
}

func NewQueueJob(loc *time.Location, qs access.QueueStorage, ms access.MemberStorage, hs access.HolidayStorage) *QueueJob {
	return &QueueJob{
		loc:            loc,
		queueStorage:   qs,
		memberStorage:  ms,
		holidayStorage: hs,
	}
}
