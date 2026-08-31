package initialize

import (
	"errors"
	"fmt"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/system"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Gorm() *gorm.DB {
	return mustOpenConfiguredDatabase()
}

func TryGorm() (db *gorm.DB, err error) {
	return openConfiguredDatabase(mustOpenConfiguredDatabase)
}

func mustOpenConfiguredDatabase() *gorm.DB {
	switch global.GVA_CONFIG.System.DbType {
	case "mysql":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Mysql.Dbname
		return GormMysql()
	case "pgsql":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Pgsql.Dbname
		return GormPgSql()
	case "oracle":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Oracle.Dbname
		return GormOracle()
	case "mssql":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Mssql.Dbname
		return GormMssql()
	case "sqlite":
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Sqlite.Dbname
		return GormSqlite()
	default:
		global.GVA_ACTIVE_DBNAME = &global.GVA_CONFIG.Mysql.Dbname
		return GormMysql()
	}
}

func openConfiguredDatabase(open func() *gorm.DB) (db *gorm.DB, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("database initialization failed: %v", r)
			db = nil
		}
	}()
	return open(), nil
}

func RegisterTables() error {
	if global.GVA_CONFIG.System.DisableAutoMigrate {
		global.GVA_LOG.Info("auto-migrate is disabled, skipping table registration")
		return nil
	}

	db := global.GVA_DB
	if db == nil {
		return errors.New("register tables: database is not initialized")
	}
	err := db.AutoMigrate(
		system.SysUser{},
		system.ExecutionRun{},
		system.SentenceGenerationRecord{},
		system.AIProviderConfig{},
		system.TTSProviderConfig{},
		system.CLIProviderConfig{},
		system.SentenceExecutorConfig{},
		// 在此添加更多基础表模型
	)

	if err != nil {
		global.GVA_LOG.Error("register table failed", zap.Error(err))
		return fmt.Errorf("register tables: %w", err)
	}

	err = registerExtraModels()

	if err != nil {
		global.GVA_LOG.Error("register extra tables failed", zap.Error(err))
		return fmt.Errorf("register extra tables: %w", err)
	}
	global.GVA_LOG.Info("register table success")
	return nil
}
