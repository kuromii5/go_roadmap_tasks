package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port string `env:"PORT" envDefault:"8080"`

	DB    DBConfig
	Redis RedisConfig
}

type DBConfig struct {
	Host     string `env:"DB_HOST"     envDefault:"localhost"`
	Port     string `env:"DB_PORT"     envDefault:"5432"`
	User     string `env:"DB_USER"     envDefault:"postgres"`
	Password string `env:"DB_PASSWORD,required"`
	Name     string `env:"DB_NAME"     envDefault:"poller"`
	SSLMode  string `env:"DB_SSLMODE"  envDefault:"disable"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR"     envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB"       envDefault:"0"`
}

func MustLoad() Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		slog.Error("parse config", "error", err)
		os.Exit(1)
	}
	return cfg
}
