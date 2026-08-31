package system

import (
	"errors"
	"strings"
	"testing"

	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func validTTSConfigPayload(apiKey string) sysModel.TTSConfigPayload {
	return sysModel.TTSConfigPayload{
		Active: "xiaomi-mimo-tts",
		Providers: []sysModel.TTSProviderPayload{{
			ProviderID: "xiaomi-mimo-tts",
			Label:      " Xiaomi MiMo TTS ",
			Type:       "mimo-tts",
			BaseURL:    " https://api.xiaomimimo.com/v1/ ",
			ApiKey:     apiKey,
			Model:      " mimo-v2.5-tts ",
			Voice:      " Chloe ",
			Enabled:    true,
		}},
	}
}

func TestNormalizeTTSConfigPreservesExistingKey(t *testing.T) {
	rows, err := normalizeTTSConfig(
		validTTSConfigPayload(""),
		map[string]storedTTSProvider{
			"xiaomi-mimo-tts": {
				apiKey:  "stored-secret",
				baseURL: "https://api.xiaomimimo.com/v1",
			},
		},
	)
	if err != nil {
		t.Fatalf("normalize config: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %d", len(rows))
	}
	if rows[0].ApiKey != "stored-secret" {
		t.Fatalf("expected stored key to be preserved, got %q", rows[0].ApiKey)
	}
	if !rows[0].Active {
		t.Fatal("expected provider to be active")
	}
	if rows[0].BaseURL != "https://api.xiaomimimo.com/v1" {
		t.Fatalf("expected normalized base URL, got %q", rows[0].BaseURL)
	}
	if rows[0].Label != "Xiaomi MiMo TTS" || rows[0].Model != "mimo-v2.5-tts" || rows[0].Voice != "Chloe" {
		t.Fatalf("expected text fields to be trimmed, got %#v", rows[0])
	}
}

func TestNormalizeTTSConfigRejectsNewProviderWithoutKey(t *testing.T) {
	_, err := normalizeTTSConfig(validTTSConfigPayload(""), nil)
	if err == nil || !strings.Contains(err.Error(), "API Key") {
		t.Fatalf("expected missing API Key error, got %v", err)
	}
}

func TestNormalizeTTSConfigRejectsDuplicateProviderID(t *testing.T) {
	input := validTTSConfigPayload("new-secret")
	input.Providers = append(input.Providers, input.Providers[0])

	_, err := normalizeTTSConfig(input, nil)
	if err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("expected duplicate ID error, got %v", err)
	}
}

func TestNormalizeTTSConfigRejectsUnsupportedType(t *testing.T) {
	input := validTTSConfigPayload("new-secret")
	input.Providers[0].Type = "openai-compatible"

	_, err := normalizeTTSConfig(input, nil)
	if err == nil || !strings.Contains(err.Error(), "类型不支持") {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestNormalizeTTSConfigRejectsNonOfficialMiMoOrigin(t *testing.T) {
	input := validTTSConfigPayload("new-secret")
	input.Providers[0].BaseURL = "https://attacker.example/v1"

	_, err := normalizeTTSConfig(input, nil)
	if err == nil || !errors.Is(err, ErrInvalidTTSConfig) {
		t.Fatalf("expected typed invalid-config error, got %v", err)
	}
	if !strings.Contains(err.Error(), "官方") {
		t.Fatalf("expected safe official-origin guidance, got %v", err)
	}
}

func TestNormalizeTTSConfigRejectsDisabledActiveProvider(t *testing.T) {
	input := validTTSConfigPayload("new-secret")
	input.Providers[0].Enabled = false

	_, err := normalizeTTSConfig(input, nil)
	if err == nil || !strings.Contains(err.Error(), "未启用") {
		t.Fatalf("expected disabled active provider error, got %v", err)
	}
}

func TestBuildSafeTTSConfigNeverReturnsAPIKey(t *testing.T) {
	response := buildSafeTTSConfig([]sysModel.TTSProviderConfig{{
		ProviderID: "xiaomi-mimo-tts",
		Label:      "Xiaomi MiMo TTS",
		Type:       "mimo-tts",
		BaseURL:    "https://api.xiaomimimo.com/v1",
		ApiKey:     "stored-secret",
		Model:      "mimo-v2.5-tts",
		Voice:      "Chloe",
		Enabled:    true,
		Active:     true,
	}})

	if response.Active != "xiaomi-mimo-tts" {
		t.Fatalf("expected active provider, got %q", response.Active)
	}
	if len(response.Providers) != 1 {
		t.Fatalf("expected one provider, got %d", len(response.Providers))
	}
	if response.Providers[0].ApiKey != "" {
		t.Fatalf("safe response exposed API Key: %q", response.Providers[0].ApiKey)
	}
	if !response.Providers[0].ApiKeyConfigured {
		t.Fatal("expected configured-key flag")
	}
}

func openTTSConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.TTSProviderConfig{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func TestTTSConfigServiceSavePreservesKeyAndReplacesStaleRows(t *testing.T) {
	db := openTTSConfigTestDB(t)
	seed := []sysModel.TTSProviderConfig{
		{
			ProviderID: "xiaomi-mimo-tts",
			Label:      "Old label",
			Type:       "mimo-tts",
			BaseURL:    "https://api.xiaomimimo.com/v1",
			ApiKey:     "stored-secret",
			Model:      "old-model",
			Voice:      "OldVoice",
			Enabled:    true,
			Active:     true,
		},
		{
			ProviderID: "stale-provider",
			Label:      "Stale",
			Type:       "mimo-tts",
			BaseURL:    "https://stale.example.com/v1",
			ApiKey:     "stale-secret",
			Model:      "stale-model",
			Voice:      "StaleVoice",
			Enabled:    false,
			Active:     false,
		},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed rows: %v", err)
	}

	service := new(TTSConfigService)
	saved, err := service.SaveConfig(db, validTTSConfigPayload(""))
	if err != nil {
		t.Fatalf("save config: %v", err)
	}
	if saved.Active != "xiaomi-mimo-tts" || len(saved.Providers) != 1 {
		t.Fatalf("unexpected safe response: %#v", saved)
	}
	if saved.Providers[0].ApiKey != "" || !saved.Providers[0].ApiKeyConfigured {
		t.Fatalf("expected masked configured key, got %#v", saved.Providers[0])
	}

	var rows []sysModel.TTSProviderConfig
	if err := db.Order("provider_id").Find(&rows).Error; err != nil {
		t.Fatalf("load persisted rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected stale row to be removed, got %d rows", len(rows))
	}
	if rows[0].ApiKey != "stored-secret" {
		t.Fatalf("expected stored key to be preserved, got %q", rows[0].ApiKey)
	}
	if rows[0].Label != "Xiaomi MiMo TTS" || rows[0].Model != "mimo-v2.5-tts" {
		t.Fatalf("expected submitted fields to be persisted, got %#v", rows[0])
	}
}

func TestTTSConfigServiceDoesNotRedirectPreservedKeyToChangedOrigin(t *testing.T) {
	db := openTTSConfigTestDB(t)
	original := sysModel.TTSProviderConfig{
		ProviderID: "xiaomi-mimo-tts",
		Label:      "Xiaomi MiMo TTS",
		Type:       "mimo-tts",
		BaseURL:    "https://api.xiaomimimo.com/v1",
		ApiKey:     "stored-secret",
		Model:      "mimo-v2.5-tts",
		Voice:      "Chloe",
		Enabled:    true,
		Active:     true,
	}
	if err := db.Create(&original).Error; err != nil {
		t.Fatalf("seed original row: %v", err)
	}

	input := validTTSConfigPayload("")
	input.Providers[0].BaseURL = "https://attacker.example/v1"
	_, err := new(TTSConfigService).SaveConfig(db, input)
	if err == nil || !errors.Is(err, ErrInvalidTTSConfig) {
		t.Fatalf("expected redirect attempt to be rejected, got %v", err)
	}

	var persisted sysModel.TTSProviderConfig
	if err := db.First(&persisted, "provider_id = ?", original.ProviderID).Error; err != nil {
		t.Fatalf("reload original row: %v", err)
	}
	if persisted.BaseURL != original.BaseURL || persisted.ApiKey != original.ApiKey {
		t.Fatalf("preserved key was redirected: %#v", persisted)
	}
}

func TestTTSConfigServiceGetConfigMasksKey(t *testing.T) {
	db := openTTSConfigTestDB(t)
	row := sysModel.TTSProviderConfig{
		ProviderID: "xiaomi-mimo-tts",
		Label:      "Xiaomi MiMo TTS",
		Type:       "mimo-tts",
		BaseURL:    "https://api.xiaomimimo.com/v1",
		ApiKey:     "stored-secret",
		Model:      "mimo-v2.5-tts",
		Voice:      "Chloe",
		Enabled:    true,
		Active:     true,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}

	loaded, err := new(TTSConfigService).GetConfig(db)
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if loaded.Active != row.ProviderID || len(loaded.Providers) != 1 {
		t.Fatalf("unexpected response: %#v", loaded)
	}
	if loaded.Providers[0].ApiKey != "" || !loaded.Providers[0].ApiKeyConfigured {
		t.Fatalf("API Key was not safely represented: %#v", loaded.Providers[0])
	}
}

func TestTTSConfigServiceSaveRollsBackWhenInsertFails(t *testing.T) {
	db := openTTSConfigTestDB(t)
	original := sysModel.TTSProviderConfig{
		ProviderID: "original-provider",
		Label:      "Original",
		Type:       "mimo-tts",
		BaseURL:    "https://original.example.com/v1",
		ApiKey:     "original-secret",
		Model:      "original-model",
		Voice:      "OriginalVoice",
		Enabled:    true,
		Active:     true,
	}
	if err := db.Create(&original).Error; err != nil {
		t.Fatalf("seed original row: %v", err)
	}
	if err := db.Exec(`
		CREATE TRIGGER reject_new_provider
		BEFORE INSERT ON tts_provider_configs
		WHEN NEW.provider_id = 'new-provider'
		BEGIN
			SELECT RAISE(ABORT, 'forced insert failure');
		END;
	`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	input := validTTSConfigPayload("new-secret")
	input.Active = "new-provider"
	input.Providers[0].ProviderID = "new-provider"
	_, err := new(TTSConfigService).SaveConfig(db, input)
	if err == nil || !errors.Is(err, ErrTTSConfigPersistence) {
		t.Fatalf("expected typed persistence failure, got %v", err)
	}

	var rows []sysModel.TTSProviderConfig
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("load rows after rollback: %v", err)
	}
	if len(rows) != 1 || rows[0].ProviderID != original.ProviderID || rows[0].ApiKey != original.ApiKey {
		t.Fatalf("expected original row after rollback, got %#v", rows)
	}
}

func TestTTSConfigServiceRejectsNilDatabase(t *testing.T) {
	service := new(TTSConfigService)
	if _, err := service.GetConfig(nil); err == nil {
		t.Fatal("expected GetConfig to reject nil database")
	}
	if _, err := service.SaveConfig(nil, validTTSConfigPayload("new-secret")); err == nil {
		t.Fatal("expected SaveConfig to reject nil database")
	}
}
