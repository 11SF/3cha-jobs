package config

import (
	"encoding/json"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Database  Database
	JobConfig string `env:"JOB_CONFIG,notEmpty"`
}

type Database struct {
	Host     string `env:"DB_HOST,notEmpty"`
	Port     string `env:"DB_PORT"     envDefault:"5432"`
	User     string `env:"DB_USER,notEmpty"`
	Password string `env:"DB_PASSWORD,notEmpty"`
	Name     string `env:"DB_NAME,notEmpty"`
	SSLMode  string `env:"DB_SSLMODE"  envDefault:"disable"`
	TimeZone string `env:"DB_TIMEZONE" envDefault:"Asia/Bangkok"`
}

// JobEntry represents one job's schedule config in jobs.json.
type JobEntry struct {
	ID     string `json:"id"`
	Hours  []int  `json:"hours"`
	Active bool   `json:"active"`
}

func (d Database) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode, d.TimeZone)
}

func Load() (Config, error) {
	var cfg Config
	return cfg, env.Parse(&cfg)
}

func ParseJobConfig(raw string) ([]JobEntry, error) {
	var entries []JobEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("parse JOB_CONFIG: %w", err)
	}
	return entries, nil
}
