package job

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Minimal model types mirroring the backend schema.

type member struct {
	ID          uuid.UUID  `gorm:"primaryKey"`
	Name        string
	AvatarColor string
	IsActive    bool
	SortOrder   int
	DeletedAt   *time.Time `gorm:"index"`
}

func (member) TableName() string { return "members" }

type dailyQueue struct {
	ID        uuid.UUID `gorm:"primaryKey"`
	QueueDate time.Time `gorm:"type:date"`
	MemberID  uuid.UUID
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (dailyQueue) TableName() string { return "daily_queue" }

type queueConfig struct {
	ID           string     `gorm:"primaryKey"`
	LastMemberID *uuid.UUID `gorm:"type:uuid"`
	UpdatedAt    time.Time
}

func (queueConfig) TableName() string { return "queue_config" }

type holiday struct {
	HolidayDate time.Time `gorm:"type:date"`
}

func (holiday) TableName() string { return "holidays" }

// DailyQueueJob generates the queue entry for the current working day.
type DailyQueueJob struct {
	db *gorm.DB
}

func NewDailyQueueJob(db *gorm.DB) *DailyQueueJob {
	return &DailyQueueJob{db: db}
}

func (j *DailyQueueJob) Run() {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	slog.Info("running daily queue job", "date", today.Format("2006-01-02"))
	if err := j.generate(today); err != nil {
		slog.Error("daily queue job failed", "error", err)
	}
}

func (j *DailyQueueJob) generate(today time.Time) error {
	// Idempotent: skip if today already has an entry.
	var existing dailyQueue
	err := j.db.Where("queue_date::date = ?", today.Format("2006-01-02")).
		Order("created_at DESC").First(&existing).Error
	if err == nil {
		slog.Info("entry already exists, skipping", "date", today.Format("2006-01-02"), "member", existing.MemberID)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if !j.isWorkingDay(today) {
		slog.Info("not a working day, skipping", "date", today.Format("2006-01-02"))
		return nil
	}

	var members []member
	if err := j.db.Where("is_active = true AND deleted_at IS NULL").
		Order("sort_order ASC, created_at ASC").Find(&members).Error; err != nil {
		return err
	}
	if len(members) == 0 {
		slog.Warn("no active members, skipping")
		return nil
	}

	// Auto-close pending entries from before today.
	j.db.Model(&dailyQueue{}).
		Where("status = ? AND queue_date::date < ?", "pending", today.Format("2006-01-02")).
		Update("status", "done")

	lastMemberID := j.lastUsedMemberID()
	next := roundRobinNext(members, lastMemberID)

	entry := dailyQueue{
		ID:        uuid.New(),
		QueueDate: today,
		MemberID:  next.ID,
		Status:    "pending",
	}
	if err := j.db.Create(&entry).Error; err != nil {
		return err
	}

	slog.Info("created queue entry", "date", today.Format("2006-01-02"), "member", next.Name)
	return nil
}

func (j *DailyQueueJob) lastUsedMemberID() string {
	var cfg queueConfig
	j.db.First(&cfg, "id = ?", "singleton")

	var lastQ dailyQueue
	err := j.db.Order("queue_date DESC, created_at DESC").First(&lastQ).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if cfg.LastMemberID != nil {
			return cfg.LastMemberID.String()
		}
		return ""
	}

	// Use config override when a reset happened after the last entry.
	if cfg.LastMemberID != nil && lastQ.CreatedAt.Before(cfg.UpdatedAt) {
		return cfg.LastMemberID.String()
	}
	return lastQ.MemberID.String()
}

func (j *DailyQueueJob) isWorkingDay(date time.Time) bool {
	wd := date.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	var h holiday
	err := j.db.Where("holiday_date::date = ?", date.Format("2006-01-02")).First(&h).Error
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func roundRobinNext(members []member, lastID string) member {
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
