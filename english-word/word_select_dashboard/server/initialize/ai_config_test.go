package initialize

import (
	"errors"
	"testing"

	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func openAIConfigSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func setAIConfigSyncGlobals(t *testing.T, db *gorm.DB, ai config.AI) *viper.Viper {
	t.Helper()
	previousDB := global.GVA_DB
	previousAI := global.GVA_CONFIG.AI
	previousVP := global.GVA_VP
	previousLog := global.GVA_LOG
	global.GVA_DB = db
	global.GVA_CONFIG.AI = ai
	global.GVA_VP = viper.New()
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_DB = previousDB
		global.GVA_CONFIG.AI = previousAI
		global.GVA_VP = previousVP
		global.GVA_LOG = previousLog
	})
	return global.GVA_VP
}

func yamlAIConfig() config.AI {
	return config.AI{Active: "yaml", Providers: map[string]config.AIProvider{
		"yaml": {
			Label: "YAML", Type: config.AIProviderTypeOpenAICompatible,
			BaseURL: "https://yaml.example/v1", ApiKey: "yaml-secret", Model: "yaml-model", MaxTokens: 4096,
		},
	}}
}

func TestSyncAIConfigWithDatabaseKeepsExistingDatabaseProviders(t *testing.T) {
	db := openAIConfigSyncTestDB(t)
	dbProvider := sysModel.AIProviderConfig{
		ProviderID: "database", Label: "Database", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://database.example/v1", ApiKey: "database-secret", Model: "database-model", MaxTokens: 2048, Active: true,
	}
	if err := db.Create(&dbProvider).Error; err != nil {
		t.Fatalf("seed database provider: %v", err)
	}
	vp := setAIConfigSyncGlobals(t, db, yamlAIConfig())

	SyncAIConfigWithDatabase()

	var rows []sysModel.AIProviderConfig
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("reload providers: %v", err)
	}
	if len(rows) != 1 || rows[0].ProviderID != "database" || rows[0].ApiKey != "database-secret" {
		t.Fatalf("YAML overwrote database providers: %#v", rows)
	}
	if global.GVA_CONFIG.AI.Active != "database" || global.GVA_CONFIG.AI.Providers["database"].ApiKey != "database-secret" {
		t.Fatalf("runtime was not loaded from database: %#v", global.GVA_CONFIG.AI)
	}
	providers, ok := vp.Get("ai.providers").(map[string]config.AIProvider)
	if !ok || providers["database"].ApiKey != "database-secret" {
		t.Fatalf("Viper memory was not loaded from database: %#v", vp.Get("ai.providers"))
	}
}

func TestSyncAIConfigWithDatabaseDoesNotSeedYAMLWhenOnlySingletonExists(t *testing.T) {
	db := openAIConfigSyncTestDB(t)
	target := sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "cli", ExecutorID: "codex-local"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}
	setAIConfigSyncGlobals(t, db, yamlAIConfig())

	SyncAIConfigWithDatabase()

	var count int64
	if err := db.Model(&sysModel.AIProviderConfig{}).Count(&count).Error; err != nil {
		t.Fatalf("count providers: %v", err)
	}
	if count != 0 {
		t.Fatalf("YAML was seeded despite existing singleton: %d rows", count)
	}
	if len(global.GVA_CONFIG.AI.Providers) != 0 || global.GVA_CONFIG.AI.Active != "" {
		t.Fatalf("stale YAML remained in runtime: %#v", global.GVA_CONFIG.AI)
	}
}

func TestSyncAIConfigWithDatabaseSeedsYAMLOnlyWhenDatabaseIsEmpty(t *testing.T) {
	db := openAIConfigSyncTestDB(t)
	setAIConfigSyncGlobals(t, db, yamlAIConfig())

	SyncAIConfigWithDatabase()

	var row sysModel.AIProviderConfig
	if err := db.First(&row, "provider_id = ?", "yaml").Error; err != nil {
		t.Fatalf("expected YAML seed: %v", err)
	}
	if !row.Active || row.ApiKey != "yaml-secret" {
		t.Fatalf("unexpected YAML seed: %#v", row)
	}
}

func TestSyncAIConfigWithDatabaseClearsStaleRuntimeWhenDatabaseLoadFails(t *testing.T) {
	db := openAIConfigSyncTestDB(t)
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "database", Label: "Database", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://database.example/v1", ApiKey: "database-secret", Model: "database-model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	vp := setAIConfigSyncGlobals(t, db, yamlAIConfig())
	callbackName := "ai_config_sync_test:fail_target_load"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == (sysModel.SentenceExecutorConfig{}).TableName() {
			tx.AddError(errors.New("forced database load failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	SyncAIConfigWithDatabase()

	if global.GVA_CONFIG.AI.Active != "" || len(global.GVA_CONFIG.AI.Providers) != 0 {
		t.Fatalf("stale YAML remained after database load failed: %#v", global.GVA_CONFIG.AI)
	}
	providers, ok := vp.Get("ai.providers").(map[string]config.AIProvider)
	if !ok || len(providers) != 0 {
		t.Fatalf("stale Viper providers remained: %#v", vp.Get("ai.providers"))
	}
}

func TestSyncAIConfigWithDatabaseClearsStaleRuntimeWhenInspectionFails(t *testing.T) {
	db := openAIConfigSyncTestDB(t)
	vp := setAIConfigSyncGlobals(t, db, yamlAIConfig())
	callbackName := "ai_config_sync_test:fail_database_inspection"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == (sysModel.AIProviderConfig{}).TableName() {
			tx.AddError(errors.New("forced database inspection failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	SyncAIConfigWithDatabase()

	if global.GVA_CONFIG.AI.Active != "" || len(global.GVA_CONFIG.AI.Providers) != 0 {
		t.Fatalf("stale YAML remained after database inspection failed: %#v", global.GVA_CONFIG.AI)
	}
	providers, ok := vp.Get("ai.providers").(map[string]config.AIProvider)
	if !ok || len(providers) != 0 {
		t.Fatalf("stale Viper providers remained: %#v", vp.Get("ai.providers"))
	}
}
