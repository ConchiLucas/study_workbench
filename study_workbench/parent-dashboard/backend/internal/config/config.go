package config

import (
	"fmt"
	"os"
	"strconv"
)

type Mastery struct {
	BaseMasterStreak int
	MinAccuracy      float64
	ShakyMinAttempts int
	ShakyAccuracy    float64
	EaseMin          float64
	EaseMax          float64
	EaseUp           float64
	EaseDown         float64
	MaxIntervalDays  int
}

// Pgsql 与同级项目（go-react-template / ai-datahub）保持同一套 Docker Postgres 约定
type Pgsql struct {
	Path     string // host，默认 127.0.0.1
	Port     string
	Config   string // sslmode=disable TimeZone=Asia/Shanghai
	Dbname   string
	Username string
	Password string
}

func (p Pgsql) Dsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s %s",
		p.Path, p.Username, p.Password, p.Dbname, p.Port, p.Config,
	)
}

type Config struct {
	Addr    string
	Driver  string // postgres | sqlite（sqlite 仅用于单测）
	DSN     string // 完整 DSN；为空时用 Pgsql 拼
	Pgsql   Pgsql
	Mastery Mastery
}

func Load() Config {
	cfg := Config{
		Addr:   env("APP_ADDR", ":19081"),
		Driver: env("APP_DB_DRIVER", "postgres"),
		DSN:    os.Getenv("APP_DSN"),
		Pgsql: Pgsql{
			Path:     env("APP_DB_HOST", "127.0.0.1"),
			Port:     env("APP_DB_PORT", "15432"),
			Config:   env("APP_DB_CONFIG", "sslmode=disable TimeZone=Asia/Shanghai"),
			Dbname:   env("APP_DB_NAME", "study_workbench"),
			Username: env("APP_DB_USER", "conchi"),
			Password: env("APP_DB_PASSWORD", "conchi123456"),
		},
		Mastery: Mastery{
			BaseMasterStreak: envInt("MASTERY_BASE_STREAK", 2),
			MinAccuracy:      0.8,
			ShakyMinAttempts: 3,
			ShakyAccuracy:    0.6,
			EaseMin:          1.3,
			EaseMax:          2.8,
			EaseUp:           0.1,
			EaseDown:         0.2,
			MaxIntervalDays:  60,
		},
	}
	if cfg.DSN == "" && cfg.Driver == "postgres" {
		cfg.DSN = cfg.Pgsql.Dsn()
	}
	return cfg
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}
