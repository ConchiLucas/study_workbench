package system

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func testAIConfig(apiKey string) config.AI {
	return config.AI{
		Active: "default",
		Providers: map[string]config.AIProvider{
			"default": {
				Label:     "Default",
				Type:      config.AIProviderTypeOpenAICompatible,
				BaseURL:   "https://api.openai.com/v1",
				ApiKey:    apiKey,
				Model:     "gpt-test",
				MaxTokens: 4096,
			},
		},
	}
}

func TestAIConfigGetMasksConfiguredAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousConfig := global.GVA_CONFIG.AI
	previousDB := global.GVA_DB
	global.GVA_CONFIG.AI = testAIConfig("stored-secret")
	global.GVA_DB = nil
	t.Cleanup(func() {
		global.GVA_CONFIG.AI = previousConfig
		global.GVA_DB = previousDB
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ai/config", nil)
	new(AIConfigApi).GetConfig(ctx)

	body := recorder.Body.String()
	if strings.Contains(body, "stored-secret") {
		t.Fatalf("AI config GET exposed API key: %s", body)
	}
	if !strings.Contains(body, `"api_key":""`) || !strings.Contains(body, `"api_key_configured":true`) {
		t.Fatalf("expected masked configured-key response, got %s", body)
	}
}

func TestNormalizeAIConfigPreservesStoredKeyOnBlankSave(t *testing.T) {
	req := AIConfigRequest{
		Active: "default",
		Providers: []AIProviderConfigItem{{
			ID:        "default",
			Label:     "Default",
			Type:      config.AIProviderTypeOpenAICompatible,
			BaseURL:   "https://api.openai.com/v1",
			ApiKey:    "",
			Model:     "gpt-test",
			MaxTokens: 4096,
		}},
	}

	req.Providers[0].BaseURL = "HTTPS://API.OPENAI.COM:443/v2"
	normalized, safeResponse, err := normalizeAIConfig(req, map[string]storedAIProvider{
		"default": {apiKey: "stored-secret", baseURL: "https://api.openai.com/v1"},
	})
	if err != nil {
		t.Fatalf("normalize AI config: %v", err)
	}
	if normalized.Providers["default"].ApiKey != "stored-secret" {
		t.Fatal("stored API key was not preserved")
	}
	if safeResponse.Providers[0].ApiKey != "" || !safeResponse.Providers[0].ApiKeyConfigured {
		t.Fatalf("save response must remain masked, got %#v", safeResponse.Providers[0])
	}
}

func TestNormalizeAIConfigRequiresKeyForNonASCIIHostnameChange(t *testing.T) {
	req := AIConfigRequest{
		Active: "default",
		Providers: []AIProviderConfigItem{{
			ID:        "default",
			Type:      config.AIProviderTypeOpenAICompatible,
			BaseURL:   "https://i.example/v2",
			Model:     "gpt-test",
			MaxTokens: 4096,
		}},
	}

	_, _, err := normalizeAIConfig(req, map[string]storedAIProvider{
		"default": {apiKey: "stored-secret", baseURL: "https://İ.example/v1"},
	})
	if err == nil || !strings.Contains(err.Error(), "服务来源已变更") {
		t.Fatalf("expected non-ASCII hostname change to require a new key, got %v", err)
	}
}

func TestNormalizeAIConfigRejectsBlankKeyWhenOriginChanges(t *testing.T) {
	req := AIConfigRequest{
		Active: "default",
		Providers: []AIProviderConfigItem{{
			ID:        "default",
			Type:      config.AIProviderTypeOpenAICompatible,
			BaseURL:   "https://attacker.example/v1",
			Model:     "gpt-test",
			MaxTokens: 4096,
		}},
	}

	_, _, err := normalizeAIConfig(req, map[string]storedAIProvider{
		"default": {apiKey: "stored-secret", baseURL: "https://api.openai.com/v1"},
	})
	if err == nil || !strings.Contains(err.Error(), "API Key") {
		t.Fatalf("expected changed origin to require a new API key, got %v", err)
	}
}

func TestNormalizeAIConfigRejectsBlankKeyForRenamedOrNewProvider(t *testing.T) {
	for _, providerID := range []string{"renamed", "new-provider"} {
		t.Run(providerID, func(t *testing.T) {
			req := AIConfigRequest{
				Active: providerID,
				Providers: []AIProviderConfigItem{{
					ID:        providerID,
					Type:      config.AIProviderTypeOpenAICompatible,
					BaseURL:   "https://api.openai.com/v1",
					Model:     "gpt-test",
					MaxTokens: 4096,
				}},
			}

			_, _, err := normalizeAIConfig(req, map[string]storedAIProvider{
				"default": {apiKey: "stored-secret", baseURL: "https://api.openai.com/v1"},
			})
			if err == nil || !strings.Contains(err.Error(), "API Key") {
				t.Fatalf("expected %s provider to require a new API key, got %v", providerID, err)
			}
		})
	}
}

func TestAIConfigSaveKeepsRuntimeInactiveForCLISentenceExecutor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := db.Create(&sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "cli", ExecutorID: "codex-local"}).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("ai: {}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	vp := viper.New()
	vp.SetConfigFile(configPath)
	if err := vp.ReadInConfig(); err != nil {
		t.Fatalf("read config: %v", err)
	}

	previousConfig := global.GVA_CONFIG.AI
	previousDB := global.GVA_DB
	previousVP := global.GVA_VP
	previousLog := global.GVA_LOG
	global.GVA_CONFIG.AI = config.AI{}
	global.GVA_DB = db
	global.GVA_VP = vp
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_CONFIG.AI = previousConfig
		global.GVA_DB = previousDB
		global.GVA_VP = previousVP
		global.GVA_LOG = previousLog
	})

	body := []byte(`{"active":"default","providers":[{"id":"default","label":"Default","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"new-secret","model":"gpt-test","max_tokens":4096}]}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/config", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	new(AIConfigApi).SaveConfig(ctx)

	if !strings.Contains(recorder.Body.String(), `"code":0`) {
		t.Fatalf("save API failed: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"active":""`) {
		t.Fatalf("save response did not reflect CLI target: %s", recorder.Body.String())
	}
	if global.GVA_CONFIG.AI.Active != "" {
		t.Fatalf("runtime activated API despite CLI singleton: %#v", global.GVA_CONFIG.AI)
	}
	var persisted sysModel.AIProviderConfig
	if err := db.First(&persisted, "provider_id = ?", "default").Error; err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if persisted.Active {
		t.Fatal("legacy API provider was active despite CLI singleton")
	}
}

func TestAIConfigSaveKeepsCommittedRuntimeWhenYAMLWriteFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := db.Create(&sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "api", ExecutorID: "default"}).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	previousConfig := global.GVA_CONFIG.AI
	previousDB := global.GVA_DB
	previousVP := global.GVA_VP
	previousLog := global.GVA_LOG
	global.GVA_CONFIG.AI = testAIConfig("stale-secret")
	global.GVA_CONFIG.AI.Providers["default"] = config.AIProvider{
		Label: "Stale", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://stale.example/v1", ApiKey: "stale-secret", Model: "stale-model", MaxTokens: 4096,
	}
	global.GVA_DB = db
	global.GVA_VP = viper.New() // WriteConfig has no target and will fail.
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_CONFIG.AI = previousConfig
		global.GVA_DB = previousDB
		global.GVA_VP = previousVP
		global.GVA_LOG = previousLog
	})
	var providerCommitted atomic.Bool
	createCallback := "ai_config_api_test:mark_provider_committed"
	queryCallback := "ai_config_api_test:reject_post_commit_read"
	if err := db.Callback().Create().After("gorm:create").Register(createCallback, func(tx *gorm.DB) {
		if tx.Statement.Table == (sysModel.AIProviderConfig{}).TableName() {
			providerCommitted.Store(true)
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register(queryCallback, func(tx *gorm.DB) {
		if providerCommitted.Load() {
			tx.AddError(errors.New("unexpected post-commit read"))
		}
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(createCallback)
		_ = db.Callback().Query().Remove(queryCallback)
	})

	body := []byte(`{"active":"default","providers":[{"id":"default","label":"Committed","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"committed-secret","model":"committed-model","max_tokens":4096}]}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/config", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	new(AIConfigApi).SaveConfig(ctx)

	if !strings.Contains(recorder.Body.String(), `"code":0`) {
		t.Fatalf("committed database save was reported as failed: %s", recorder.Body.String())
	}
	provider := global.GVA_CONFIG.AI.Providers["default"]
	if global.GVA_CONFIG.AI.Active != "default" || provider.Model != "committed-model" || provider.ApiKey != "committed-secret" {
		t.Fatalf("runtime did not reflect committed database config: %#v", global.GVA_CONFIG.AI)
	}
}

func TestAIConfigGetRefreshesUsableStaleRuntimeFromDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "database", Label: "Database", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://database.example/v1", ApiKey: "database-secret", Model: "database-model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed database provider: %v", err)
	}

	previousConfig := global.GVA_CONFIG.AI
	previousDB := global.GVA_DB
	previousVP := global.GVA_VP
	previousLog := global.GVA_LOG
	global.GVA_CONFIG.AI = config.AI{Active: "stale", Providers: map[string]config.AIProvider{
		"stale": {Label: "Stale", Type: config.AIProviderTypeOpenAICompatible, BaseURL: "https://stale.example/v1", ApiKey: "stale-secret", Model: "stale-model"},
	}}
	global.GVA_DB = db
	global.GVA_VP = viper.New()
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_CONFIG.AI = previousConfig
		global.GVA_DB = previousDB
		global.GVA_VP = previousVP
		global.GVA_LOG = previousLog
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ai/config", nil)
	new(AIConfigApi).GetConfig(ctx)

	if !strings.Contains(recorder.Body.String(), `"id":"database"`) || strings.Contains(recorder.Body.String(), `"id":"stale"`) {
		t.Fatalf("GET did not refresh from database: %s", recorder.Body.String())
	}
	if global.GVA_CONFIG.AI.Active != "database" {
		t.Fatalf("runtime cache was not refreshed: %#v", global.GVA_CONFIG.AI)
	}
}

func TestAIConfigGetFailsWhenDatabaseRefreshFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "database", Label: "Database", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://database.example/v1", ApiKey: "database-secret", Model: "database-model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed database provider: %v", err)
	}
	callbackName := "ai_config_api_test:fail_database_refresh"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == (sysModel.SentenceExecutorConfig{}).TableName() {
			tx.AddError(errors.New("forced refresh failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	previousConfig := global.GVA_CONFIG.AI
	previousDB := global.GVA_DB
	previousVP := global.GVA_VP
	previousLog := global.GVA_LOG
	global.GVA_CONFIG.AI = testAIConfig("stale-secret")
	global.GVA_DB = db
	global.GVA_VP = viper.New()
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_CONFIG.AI = previousConfig
		global.GVA_DB = previousDB
		global.GVA_VP = previousVP
		global.GVA_LOG = previousLog
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ai/config", nil)
	new(AIConfigApi).GetConfig(ctx)

	if !strings.Contains(recorder.Body.String(), `"code":7`) || strings.Contains(recorder.Body.String(), `"model":"gpt-test"`) {
		t.Fatalf("GET returned stale success after database failure: %s", recorder.Body.String())
	}
}

func TestAIConfigGetUsesOneSnapshotAcrossConcurrentPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}, &sysModel.SentenceExecutorConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "version-a", Label: "Version A", Type: config.AIProviderTypeOpenAICompatible,
		BaseURL: "https://a.example/v1", ApiKey: "secret-a", Model: "model-a", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed version A: %v", err)
	}

	previousConfig := global.GVA_CONFIG.AI
	previousDB := global.GVA_DB
	previousVP := global.GVA_VP
	previousLog := global.GVA_LOG
	global.GVA_CONFIG.AI = config.AI{}
	global.GVA_DB = db
	global.GVA_VP = viper.New()
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_CONFIG.AI = previousConfig
		global.GVA_DB = previousDB
		global.GVA_VP = previousVP
		global.GVA_LOG = previousLog
	})

	snapshotReady := make(chan struct{})
	releaseGet := make(chan struct{})
	var hookOnce sync.Once
	aiConfigGetSnapshotHook = func() {
		hookOnce.Do(func() { close(snapshotReady) })
		<-releaseGet
	}
	t.Cleanup(func() { aiConfigGetSnapshotHook = nil })

	getRecorder := httptest.NewRecorder()
	getDone := make(chan struct{})
	go func() {
		ctx, _ := gin.CreateTestContext(getRecorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ai/config", nil)
		new(AIConfigApi).GetConfig(ctx)
		close(getDone)
	}()
	select {
	case <-snapshotReady:
	case <-time.After(5 * time.Second):
		t.Fatal("GET did not capture snapshot")
	}

	postBody := []byte(`{"active":"version-b","providers":[{"id":"version-b","label":"Version B","type":"openai-compatible","base_url":"https://b.example/v1","api_key":"secret-b","model":"model-b","max_tokens":4096}]}`)
	postRecorder := httptest.NewRecorder()
	postCtx, _ := gin.CreateTestContext(postRecorder)
	postCtx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/config", bytes.NewReader(postBody))
	postCtx.Request.Header.Set("Content-Type", "application/json")
	new(AIConfigApi).SaveConfig(postCtx)
	if !strings.Contains(postRecorder.Body.String(), `"code":0`) {
		close(releaseGet)
		t.Fatalf("POST version B failed: %s", postRecorder.Body.String())
	}
	close(releaseGet)
	select {
	case <-getDone:
	case <-time.After(5 * time.Second):
		t.Fatal("GET did not finish")
	}

	body := getRecorder.Body.String()
	if !strings.Contains(body, `"active":"version-a"`) || !strings.Contains(body, `"id":"version-a"`) || strings.Contains(body, `version-b`) {
		t.Fatalf("GET response mixed snapshots: %s", body)
	}
}
