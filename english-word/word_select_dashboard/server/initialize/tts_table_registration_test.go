package initialize

import (
	"context"
	"testing"

	"github.com/conchi/go-react-template/server/global"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func openTTSTableRegistrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}

func TestEnsureTablesCreatesTTSProviderConfigTable(t *testing.T) {
	db := openTTSTableRegistrationTestDB(t)
	ctx := context.WithValue(context.Background(), "db", db)

	if _, err := new(ensureTables).MigrateTable(ctx); err != nil {
		t.Fatalf("migrate ensured tables: %v", err)
	}
	if !db.Migrator().HasTable(&sysModel.TTSProviderConfig{}) {
		t.Fatal("ensureTables did not create tts_provider_configs")
	}
	if !new(ensureTables).TableCreated(ctx) {
		t.Fatal("ensureTables did not report all required tables as created")
	}
}

func TestEnsureTablesPropagatesAutoMigrateFailure(t *testing.T) {
	db := openTTSTableRegistrationTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql database: %v", err)
	}
	ctx := context.WithValue(context.Background(), "db", db)

	if _, err := new(ensureTables).MigrateTable(ctx); err == nil {
		t.Fatal("expected AutoMigrate failure to be propagated")
	}
}

func TestRegisterTablesCreatesTTSProviderConfigTable(t *testing.T) {
	db := openTTSTableRegistrationTestDB(t)
	previousDB := global.GVA_DB
	previousLog := global.GVA_LOG
	previousDisabled := global.GVA_CONFIG.System.DisableAutoMigrate
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	global.GVA_CONFIG.System.DisableAutoMigrate = false
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		global.GVA_LOG = previousLog
		global.GVA_CONFIG.System.DisableAutoMigrate = previousDisabled
	})

	if err := RegisterTables(); err != nil {
		t.Fatalf("register tables: %v", err)
	}

	if !db.Migrator().HasTable(&sysModel.TTSProviderConfig{}) {
		t.Fatal("RegisterTables did not create tts_provider_configs")
	}
}

func TestRegisterTablesPropagatesAutoMigrateFailure(t *testing.T) {
	db := openTTSTableRegistrationTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql database: %v", err)
	}
	previousDB := global.GVA_DB
	previousLog := global.GVA_LOG
	previousDisabled := global.GVA_CONFIG.System.DisableAutoMigrate
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	global.GVA_CONFIG.System.DisableAutoMigrate = false
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		global.GVA_LOG = previousLog
		global.GVA_CONFIG.System.DisableAutoMigrate = previousDisabled
	})

	if err := RegisterTables(); err == nil {
		t.Fatal("expected RegisterTables to propagate AutoMigrate failure")
	}
}
