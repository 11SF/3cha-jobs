package access

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Member struct {
	bun.BaseModel `bun:"table:members"`

	ID          uuid.UUID  `bun:",pk,type:uuid"`
	Name        string
	AvatarColor string
	IsActive    bool
	SortOrder   int
	DeletedAt   *time.Time `bun:",nullzero"`
}

type DailyQueue struct {
	bun.BaseModel `bun:"table:daily_queue"`

	ID        uuid.UUID `bun:",pk,type:uuid"`
	QueueDate time.Time `bun:"type:date"`
	MemberID  uuid.UUID `bun:"type:uuid"`
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type QueueConfig struct {
	bun.BaseModel `bun:"table:queue_config"`

	ID           string     `bun:",pk"`
	LastMemberID *uuid.UUID `bun:"type:uuid,nullzero"`
	UpdatedAt    time.Time
}

type Holiday struct {
	bun.BaseModel `bun:"table:holidays"`

	HolidayDate time.Time `bun:"type:date"`
}
