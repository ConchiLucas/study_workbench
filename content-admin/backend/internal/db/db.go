package db

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func DSNFromEnv() string {
	if v := os.Getenv("APP_DSN"); v != "" {
		return v
	}
	host := env("APP_DB_HOST", "127.0.0.1")
	port := env("APP_DB_PORT", "15432")
	user := env("APP_DB_USER", "conchi")
	pass := env("APP_DB_PASSWORD", "conchi123456")
	name := env("APP_DB_NAME", "study_workbench")
	cfg := env("APP_DB_CONFIG", "sslmode=disable")
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s %s",
		host, user, pass, name, port, cfg)
}

func OpenPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
}

func OpenSQLite(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		return nil, err
	}
	if err := gdb.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, err
	}
	return gdb, nil
}

func Migrate(gdb *gorm.DB) error {
	sql := `
CREATE TABLE IF NOT EXISTS literacy_assets (
  kp_id BIGINT PRIMARY KEY,
  char_text VARCHAR(8) NOT NULL,
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		sql = `
CREATE TABLE IF NOT EXISTS literacy_assets (
  kp_id INTEGER PRIMARY KEY,
  char_text TEXT NOT NULL,
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(sql).Error; err != nil {
		return err
	}
	// Existing DBs: add speech_audio_url if missing.
	if gdb.Dialector.Name() == "sqlite" {
		_ = gdb.Exec(`ALTER TABLE literacy_assets ADD COLUMN speech_audio_url TEXT NOT NULL DEFAULT ''`).Error
	} else {
		_ = gdb.Exec(`ALTER TABLE literacy_assets ADD COLUMN IF NOT EXISTS speech_audio_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
	}

	pinyinSQL := `
CREATE TABLE IF NOT EXISTS pinyin_assets (
  kp_id BIGINT PRIMARY KEY,
  letter VARCHAR(16) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  solo_text VARCHAR(16) NOT NULL DEFAULT '',
  word_text VARCHAR(16) NOT NULL DEFAULT '',
  solo_speech_url VARCHAR(512) NOT NULL DEFAULT '',
  word_speech_url VARCHAR(512) NOT NULL DEFAULT '',
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		pinyinSQL = `
CREATE TABLE IF NOT EXISTS pinyin_assets (
  kp_id INTEGER PRIMARY KEY,
  letter TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  solo_text TEXT NOT NULL DEFAULT '',
  word_text TEXT NOT NULL DEFAULT '',
  solo_speech_url TEXT NOT NULL DEFAULT '',
  word_speech_url TEXT NOT NULL DEFAULT '',
  glyph_image_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(pinyinSQL).Error; err != nil {
		return err
	}
	// Existing DBs: add glyph_image_url if missing.
	if gdb.Dialector.Name() == "sqlite" {
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN glyph_image_url TEXT NOT NULL DEFAULT ''`).Error
	} else {
		_ = gdb.Exec(`ALTER TABLE pinyin_assets ADD COLUMN IF NOT EXISTS glyph_image_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
	}

	mathSQL := `
CREATE TABLE IF NOT EXISTS math_assets (
  kp_id BIGINT PRIMARY KEY,
  title VARCHAR(80) NOT NULL DEFAULT '',
  kind VARCHAR(24) NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}',
  difficulty INT NOT NULL DEFAULT 1,
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_text VARCHAR(120) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		mathSQL = `
CREATE TABLE IF NOT EXISTS math_assets (
  kp_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}',
  difficulty INTEGER NOT NULL DEFAULT 1,
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  speech_text TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(mathSQL).Error; err != nil {
		return err
	}
	if gdb.Dialector.Name() == "sqlite" {
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN glyph_image_url TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN speech_audio_url TEXT NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN speech_text TEXT NOT NULL DEFAULT ''`).Error
	} else {
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS glyph_image_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS speech_audio_url VARCHAR(512) NOT NULL DEFAULT ''`).Error
		_ = gdb.Exec(`ALTER TABLE math_assets ADD COLUMN IF NOT EXISTS speech_text VARCHAR(120) NOT NULL DEFAULT ''`).Error
	}

	englishSQL := `
CREATE TABLE IF NOT EXISTS english_assets (
  kp_id BIGINT PRIMARY KEY,
  word_text VARCHAR(32) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		englishSQL = `
CREATE TABLE IF NOT EXISTS english_assets (
  kp_id INTEGER PRIMARY KEY,
  word_text TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(englishSQL).Error; err != nil {
		return err
	}

	scienceSQL := `
CREATE TABLE IF NOT EXISTS science_assets (
  kp_id BIGINT PRIMARY KEY,
  title VARCHAR(64) NOT NULL DEFAULT '',
  module_code VARCHAR(50) NOT NULL DEFAULT '',
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  module_order INT NOT NULL DEFAULT 0,
  kp_order INT NOT NULL DEFAULT 0,
  needs_sense_image BOOLEAN NOT NULL,
  needs_sense_image_override BOOLEAN,
  glyph_image_url VARCHAR(512) NOT NULL DEFAULT '',
  sense_image_url VARCHAR(512) NOT NULL DEFAULT '',
  speech_audio_url VARCHAR(512) NOT NULL DEFAULT '',
  synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`
	if gdb.Dialector.Name() == "sqlite" {
		scienceSQL = `
CREATE TABLE IF NOT EXISTS science_assets (
  kp_id INTEGER PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  module_code TEXT NOT NULL DEFAULT '',
  module_name TEXT NOT NULL DEFAULT '',
  module_order INTEGER NOT NULL DEFAULT 0,
  kp_order INTEGER NOT NULL DEFAULT 0,
  needs_sense_image INTEGER NOT NULL,
  needs_sense_image_override INTEGER,
  glyph_image_url TEXT NOT NULL DEFAULT '',
  sense_image_url TEXT NOT NULL DEFAULT '',
  speech_audio_url TEXT NOT NULL DEFAULT '',
  synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	}
	if err := gdb.Exec(scienceSQL).Error; err != nil {
		return err
	}

	qtaskSQL := `
CREATE TABLE IF NOT EXISTS question_tasks (
  id BIGSERIAL PRIMARY KEY,
  subject_code VARCHAR(30) NOT NULL,
  title VARCHAR(80) NOT NULL,
  module_code VARCHAR(50) NOT NULL,
  module_name VARCHAR(50) NOT NULL DEFAULT '',
  target_count INT NOT NULL DEFAULT 10,
  status VARCHAR(20) NOT NULL DEFAULT 'draft',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_question_tasks_subject_status ON question_tasks(subject_code, status);
CREATE INDEX IF NOT EXISTS idx_question_tasks_module ON question_tasks(module_code);

CREATE TABLE IF NOT EXISTS question_task_items (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL REFERENCES question_tasks(id) ON DELETE CASCADE,
  seq INT NOT NULL,
  kp_id BIGINT NOT NULL,
  question_id BIGINT NOT NULL,
  UNIQUE(task_id, seq),
  UNIQUE(task_id, question_id)
);`
	if gdb.Dialector.Name() == "sqlite" {
		qtaskSQL = `
CREATE TABLE IF NOT EXISTS question_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  subject_code TEXT NOT NULL,
  title TEXT NOT NULL,
  module_code TEXT NOT NULL,
  module_name TEXT NOT NULL DEFAULT '',
  target_count INTEGER NOT NULL DEFAULT 10,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_question_tasks_subject_status ON question_tasks(subject_code, status);
CREATE INDEX IF NOT EXISTS idx_question_tasks_module ON question_tasks(module_code);

CREATE TABLE IF NOT EXISTS question_task_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  task_id INTEGER NOT NULL REFERENCES question_tasks(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  kp_id INTEGER NOT NULL,
  question_id INTEGER NOT NULL,
  UNIQUE(task_id, seq),
  UNIQUE(task_id, question_id)
);`
	}
	if err := gdb.Exec(qtaskSQL).Error; err != nil {
		return err
	}

	return nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
