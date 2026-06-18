package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"portal/batch/config"
	"portal/batch/job/queue"
	"portal/batch/job/queue/access"
	"portal/batch/scheduler"
)

var commit string

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		panic(err)
	}

	db, err := connectDB(cfg.Database.DSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		panic(err)
	}

	bkk, _ := time.LoadLocation("Asia/Bangkok")

	s := registerJobs(cfg, db, bkk)
	if err := s.Validate(); err != nil {
		slog.Error("job config validation failed", "error", err)
		panic(err)
	}

	now := time.Now().In(bkk)
	slog.Info("batch triggered", "time", now.Format("2006-01-02 15:04:05"))

	if err := s.Run(ctx, now); err != nil {
		slog.Error("failed to run scheduler", "error", err)
		panic(err)
	}
}

func registerJobs(cfg config.Config, db *bun.DB, loc *time.Location) *scheduler.Scheduler {
	jobEntries, err := config.ParseJobConfig(cfg.JobConfig)
	if err != nil {
		slog.Error("failed to load job config", "error", err)
		panic(err)
	}

	s := scheduler.New(jobEntries)
	{
		qs := access.NewQueueStorage(db)
		hs := access.NewHolidayStorage(db)
		ms := access.NewMemberStorage(db)
		s.Register(queue.NewQueueJob(loc, qs, ms, hs))
		s.Register(queue.NewMarkQueueDoneJob(loc, qs, hs))
	}

	return s
}

func connectDB(dsn string) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	sqldb.SetMaxIdleConns(2)
	sqldb.SetMaxOpenConns(5)
	sqldb.SetConnMaxLifetime(time.Hour)

	for i := range 10 {
		if err := db.Ping(); err == nil {
			break
		}
		slog.Warn("waiting for database...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}
	return db, db.Ping()
}
