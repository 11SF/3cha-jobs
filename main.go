package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"portal/batch/config"
	"portal/batch/job"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := connectDB(cfg.Database.DSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	runner := job.NewDailyQueueJob(db)

	// Run immediately on startup to catch any missed entry (e.g. after restart).
	runner.Run()

	// Schedule to run at 00:05 UTC every day.
	c := cron.New()
	if _, err := c.AddFunc("5 0 * * *", runner.Run); err != nil {
		slog.Error("failed to register cron", "error", err)
		os.Exit(1)
	}
	c.Start()

	slog.Info("batch service started — scheduled at 00:05 UTC daily")
	select {} // block forever
}

func connectDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Wait for the DB to be ready (useful in Docker Compose startup race).
	for i := 0; i < 10; i++ {
		if err := sqlDB.Ping(); err == nil {
			break
		}
		slog.Warn("waiting for database...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}
	return db, sqlDB.Ping()
}

