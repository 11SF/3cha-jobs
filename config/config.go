package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Database Database
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

func (d Database) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode, d.TimeZone)
}

func Load() (Config, error) {
	var cfg Config
	return cfg, env.Parse(&cfg)
}
