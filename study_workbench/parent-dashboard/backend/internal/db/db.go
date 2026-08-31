package db

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/sqlite/*.sql migrations/postgres/*.sql
var migrationFS embed.FS

const (
	DriverPostgres = "postgres"
	DriverSQLite   = "sqlite"
)

func Open(driver, dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}

	switch driver {
	case DriverPostgres, "pgsql", "postgresql":
		return gorm.Open(postgres.Open(dsn), cfg)
	case DriverSQLite, "":
		gdb, err := gorm.Open(sqlite.Open(dsn), cfg)
		if err != nil {
			return nil, err
		}
		if err := gdb.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			return nil, err
		}
		return gdb, nil
	default:
		return nil, fmt.Errorf("unsupported db driver %q", driver)
	}
}

// OpenMemory 单测用内存 SQLite
func OpenMemory() (*gorm.DB, error) {
	return Open(DriverSQLite, ":memory:")
}

func Migrate(gdb *gorm.DB) error {
	driver := DriverSQLite
	metaSQL := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY, applied_at DATETIME NOT NULL)`
	if gdb.Dialector.Name() == "postgres" {
		driver = DriverPostgres
		metaSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL)`
	}

	if err := gdb.Exec(metaSQL).Error; err != nil {
		return err
	}

	pattern := fmt.Sprintf("migrations/%s/*.sql", driver)
	files, err := fs.Glob(migrationFS, pattern)
	if err != nil {
		return err
	}
	sort.Strings(files)

	prefix := fmt.Sprintf("migrations/%s/", driver)
	for _, f := range files {
		version := strings.TrimPrefix(f, prefix)

		var n int64
		if err := gdb.Raw("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).
			Scan(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}

		body, err := migrationFS.ReadFile(f)
		if err != nil {
			return err
		}

		err = gdb.Transaction(func(tx *gorm.DB) error {
			for _, stmt := range strings.Split(string(body), ";") {
				if strings.TrimSpace(stmt) == "" {
					continue
				}
				if err := tx.Exec(stmt).Error; err != nil {
					return fmt.Errorf("%s/%s: %w", driver, version, err)
				}
			}
			return tx.Exec("INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)",
				version, time.Now()).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}
