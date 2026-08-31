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

func openSentenceExecutorTableRegistrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	return db
}

func assertSentenceExecutorSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	if !db.Migrator().HasTable(&sysModel.CLIProviderConfig{}) {
		t.Fatal("migration did not create cli_provider_configs")
	}
	if !db.Migrator().HasTable(&sysModel.SentenceExecutorConfig{}) {
		t.Fatal("migration did not create sentence_executor_config")
	}

	columnTypes, err := db.Migrator().ColumnTypes(&sysModel.SentenceExecutorConfig{})
	if err != nil {
		t.Fatalf("inspect sentence executor columns: %v", err)
	}
	var singletonPrimaryKey bool
	for _, columnType := range columnTypes {
		if columnType.Name() == "singleton_key" {
			singletonPrimaryKey, _ = columnType.PrimaryKey()
			break
		}
	}
	if !singletonPrimaryKey {
		t.Fatal("sentence_executor_config.singleton_key is not a primary key")
	}

	indexes, err := db.Migrator().GetIndexes(&sysModel.CLIProviderConfig{})
	if err != nil {
		t.Fatalf("inspect CLI provider indexes: %v", err)
	}
	var providerIDUnique bool
	for _, index := range indexes {
		unique, ok := index.Unique()
		if !ok || !unique {
			continue
		}
		for _, column := range index.Columns() {
			if column == "provider_id" {
				providerIDUnique = true
				break
			}
		}
	}
	if !providerIDUnique {
		t.Fatal("cli_provider_configs.provider_id does not have a unique index")
	}
}

func TestEnsureTablesCreatesSentenceExecutorSchema(t *testing.T) {
	db := openSentenceExecutorTableRegistrationTestDB(t)
	ctx := context.WithValue(context.Background(), "db", db)

	if _, err := new(ensureTables).MigrateTable(ctx); err != nil {
		t.Fatalf("migrate ensured tables: %v", err)
	}
	assertSentenceExecutorSchema(t, db)
	if !new(ensureTables).TableCreated(ctx) {
		t.Fatal("ensureTables did not report all required tables as created")
	}
}

func TestRegisterTablesCreatesSentenceExecutorSchema(t *testing.T) {
	db := openSentenceExecutorTableRegistrationTestDB(t)
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
	assertSentenceExecutorSchema(t, db)
}

func TestCLIProviderConfigPreservesDisabledOnCreate(t *testing.T) {
	db := openSentenceExecutorTableRegistrationTestDB(t)
	ctx := context.WithValue(context.Background(), "db", db)
	if _, err := new(ensureTables).MigrateTable(ctx); err != nil {
		t.Fatalf("migrate ensured tables: %v", err)
	}

	disabled := false
	config := sysModel.CLIProviderConfig{
		ProviderID:       "disabled-cli",
		Label:            "Disabled CLI",
		Driver:           "codex",
		CommandPath:      "/usr/local/bin/codex",
		Model:            "gpt-5",
		WorkingDirectory: "/tmp",
		TimeoutSeconds:   300,
		Enabled:          &disabled,
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("create disabled CLI provider: %v", err)
	}

	var stored sysModel.CLIProviderConfig
	if err := db.First(&stored, config.ID).Error; err != nil {
		t.Fatalf("reload disabled CLI provider: %v", err)
	}
	if stored.Enabled == nil {
		t.Fatal("disabled CLI provider reloaded without an enabled value")
	}
	if *stored.Enabled {
		t.Fatal("disabled CLI provider was persisted as enabled")
	}
}

func TestCLIProviderConfigDefaultsEnabledWhenOmitted(t *testing.T) {
	db := openSentenceExecutorTableRegistrationTestDB(t)
	ctx := context.WithValue(context.Background(), "db", db)
	if _, err := new(ensureTables).MigrateTable(ctx); err != nil {
		t.Fatalf("migrate ensured tables: %v", err)
	}

	result := db.Exec(`
		INSERT INTO cli_provider_configs
			(provider_id, label, driver, command_path, model, working_directory)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "default-enabled-cli", "Default Enabled CLI", "codex", "/usr/local/bin/codex", "gpt-5", "/tmp")
	if result.Error != nil {
		t.Fatalf("insert CLI provider without enabled: %v", result.Error)
	}

	var storedEnabled bool
	query := db.Model(&sysModel.CLIProviderConfig{}).
		Select("enabled").
		Where("provider_id = ?", "default-enabled-cli").
		Scan(&storedEnabled)
	if query.Error != nil {
		t.Fatalf("reload default-enabled CLI provider: %v", query.Error)
	}
	if query.RowsAffected != 1 {
		t.Fatalf("reload default-enabled CLI provider: got %d rows", query.RowsAffected)
	}
	if !storedEnabled {
		t.Fatal("CLI provider without enabled did not use the database default true")
	}
}
