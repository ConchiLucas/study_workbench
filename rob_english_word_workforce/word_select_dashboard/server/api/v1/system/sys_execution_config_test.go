package system

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	sysService "github.com/conchi/go-react-template/server/service/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openExecutionConfigAPITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&sysModel.AIProviderConfig{},
		&sysModel.CLIProviderConfig{},
		&sysModel.SentenceExecutorConfig{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func useExecutionConfigAPITestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })
}

func decodeExecutionConfigResponse(t *testing.T, recorder *httptest.ResponseRecorder) (response.Response, ExecutionConfigResponse) {
	t.Helper()
	var envelope response.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response envelope: %v; body=%s", err, recorder.Body.String())
	}
	data, err := json.Marshal(envelope.Data)
	if err != nil {
		t.Fatalf("re-encode response data: %v", err)
	}
	var config ExecutionConfigResponse
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decode execution config: %v; data=%s", err, data)
	}
	return envelope, config
}

func TestBuildExecutionConfigResponseMasksKeysWithoutMutatingInput(t *testing.T) {
	input := sysService.ExecutionConfigInput{
		ActiveTarget: sysService.ExecutionTarget{Type: "api", ID: "configured"},
		APIProviders: []sysService.AIProviderInput{
			{ID: "configured", Label: "Configured", APIKey: "secret"},
			{ID: "blank", Label: "Blank", APIKey: "  "},
		},
		CLIProviders: []sysService.CLIProviderInput{{ID: "codex-local", Label: "Codex Local"}},
	}

	result := buildExecutionConfigResponse(input)
	var _ []ExecutionCLIProviderResponse = result.CLIProviders

	if result.APIProviders[0].APIKey != "" || !result.APIProviders[0].APIKeyConfigured {
		t.Fatalf("configured key was not masked: %#v", result.APIProviders[0])
	}
	if result.APIProviders[1].APIKey != "" || result.APIProviders[1].APIKeyConfigured {
		t.Fatalf("blank key had wrong configured state: %#v", result.APIProviders[1])
	}
	if input.APIProviders[0].APIKey != "secret" || input.APIProviders[1].APIKey != "  " {
		t.Fatalf("input API providers were mutated: %#v", input.APIProviders)
	}
	result.CLIProviders[0].Label = "Changed"
	if input.CLIProviders[0].Label != "Codex Local" {
		t.Fatalf("input CLI providers shared response storage: %#v", input.CLIProviders)
	}
}

func TestExecutionConfigAPIGetReturnsMaskedUnifiedConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openExecutionConfigAPITestDB(t)
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "openai", Label: "OpenAI", Type: "openai-compatible",
		BaseURL: "https://api.openai.com/v1", ApiKey: "get-secret", Model: "gpt-test", MaxTokens: 2048,
	}).Error; err != nil {
		t.Fatalf("seed API provider: %v", err)
	}
	enabled := true
	if err := db.Create(&sysModel.CLIProviderConfig{
		ProviderID: "codex-local", Label: "Codex Local", Driver: "codex",
		CommandPath: "/usr/local/bin/codex", Model: "gpt-5.6-sol", ReasoningEffort: "high",
		WorkingDirectory: "/tmp/workspace", TimeoutSeconds: 300, Enabled: &enabled,
	}).Error; err != nil {
		t.Fatalf("seed CLI provider: %v", err)
	}
	if err := db.Create(&sysModel.SentenceExecutorConfig{
		SingletonKey: "default", ExecutorType: "cli", ExecutorID: "codex-local",
	}).Error; err != nil {
		t.Fatalf("seed active target: %v", err)
	}
	useExecutionConfigAPITestDB(t, db)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ai/execution-config", nil)
	new(ExecutionConfigApi).GetConfig(ctx)

	envelope, config := decodeExecutionConfigResponse(t, recorder)
	if envelope.Code != response.SUCCESS {
		t.Fatalf("GET failed: %#v; body=%s", envelope, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "get-secret") {
		t.Fatalf("GET response exposed API key: %s", recorder.Body.String())
	}
	if config.ActiveTarget != (sysService.ExecutionTarget{Type: "cli", ID: "codex-local"}) {
		t.Fatalf("wrong active target: %#v", config.ActiveTarget)
	}
	if len(config.APIProviders) != 1 || config.APIProviders[0].APIKey != "" || !config.APIProviders[0].APIKeyConfigured {
		t.Fatalf("wrong masked API providers: %#v", config.APIProviders)
	}
	if len(config.CLIProviders) != 1 || config.CLIProviders[0].ID != "codex-local" || !config.CLIProviders[0].Enabled {
		t.Fatalf("wrong CLI providers: %#v", config.CLIProviders)
	}
}

func TestExecutionConfigAPIPostSavesAndReturnsMaskedConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("new key", func(t *testing.T) {
		db := openExecutionConfigAPITestDB(t)
		useExecutionConfigAPITestDB(t, db)
		body := `{
			"active_target":{"type":"api","id":"openai"},
			"api_providers":[{"id":"openai","label":"OpenAI","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"post-secret","model":"gpt-test","max_tokens":2048}],
			"cli_providers":[{"id":"codex-local","label":"Codex Local","driver":"codex","command_path":"/usr/local/bin/codex","model":"gpt-5.6-sol","reasoning_effort":"high","working_directory":"/tmp/workspace","timeout_seconds":300,"enabled":true}]
		}`
		recorder := invokeExecutionConfigSave(t, body)
		envelope, config := decodeExecutionConfigResponse(t, recorder)
		if envelope.Code != response.SUCCESS {
			t.Fatalf("POST failed: %#v; body=%s", envelope, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "post-secret") || config.APIProviders[0].APIKey != "" || !config.APIProviders[0].APIKeyConfigured {
			t.Fatalf("POST response exposed or misreported API key: %s", recorder.Body.String())
		}
		var target sysModel.SentenceExecutorConfig
		if err := db.First(&target, "singleton_key = ?", "default").Error; err != nil {
			t.Fatalf("load persisted target: %v", err)
		}
		if target.ExecutorType != "api" || target.ExecutorID != "openai" {
			t.Fatalf("wrong persisted target: %#v", target)
		}
		var provider sysModel.AIProviderConfig
		if err := db.First(&provider, "provider_id = ?", "openai").Error; err != nil {
			t.Fatalf("load persisted provider: %v", err)
		}
		if provider.ApiKey != "post-secret" || provider.Model != "gpt-test" || !provider.Active {
			t.Fatalf("wrong persisted provider: %#v", provider)
		}
	})

	t.Run("blank same-origin key", func(t *testing.T) {
		db := openExecutionConfigAPITestDB(t)
		if err := db.Create(&sysModel.AIProviderConfig{
			ProviderID: "openai", Label: "Old", Type: "openai-compatible",
			BaseURL: "https://api.openai.com/v1", ApiKey: "stored-secret", Model: "old", MaxTokens: 1024, Active: true,
		}).Error; err != nil {
			t.Fatalf("seed API provider: %v", err)
		}
		useExecutionConfigAPITestDB(t, db)
		body := `{
			"active_target":{"type":"api","id":"openai"},
			"api_providers":[{"id":"openai","label":"OpenAI","type":"openai-compatible","base_url":"https://api.openai.com:443/v2","api_key":"","model":"gpt-new","max_tokens":4096}],
			"cli_providers":[]
		}`
		recorder := invokeExecutionConfigSave(t, body)
		envelope, config := decodeExecutionConfigResponse(t, recorder)
		if envelope.Code != response.SUCCESS {
			t.Fatalf("blank-key POST failed: %#v; body=%s", envelope, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "stored-secret") || config.APIProviders[0].APIKey != "" || !config.APIProviders[0].APIKeyConfigured {
			t.Fatalf("blank-key response exposed or misreported API key: %s", recorder.Body.String())
		}
		var provider sysModel.AIProviderConfig
		if err := db.First(&provider, "provider_id = ?", "openai").Error; err != nil {
			t.Fatalf("load persisted provider: %v", err)
		}
		if provider.ApiKey != "stored-secret" || provider.Model != "gpt-new" {
			t.Fatalf("blank save did not preserve key and update config: %#v", provider)
		}
	})
}

func invokeExecutionConfigSave(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	return invokeExecutionConfigSaveWithContentType(t, body, "application/json")
}

func invokeExecutionConfigSaveWithContentType(t *testing.T, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/ai/execution-config", bytes.NewBufferString(body))
	if contentType != "" {
		ctx.Request.Header.Set("Content-Type", contentType)
	}
	new(ExecutionConfigApi).SaveConfig(ctx)
	return recorder
}

func validExecutionConfigRequestJSON(apiKey string) string {
	return `{
		"active_target":{"type":"api","id":"openai"},
		"api_providers":[{"id":"openai","label":"OpenAI","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"` + apiKey + `","model":"gpt-test","max_tokens":2048}],
		"cli_providers":[]
	}`
}

func TestExecutionConfigAPIFailuresDoNotExposeSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("invalid JSON", func(t *testing.T) {
		useExecutionConfigAPITestDB(t, openExecutionConfigAPITestDB(t))
		recorder := invokeExecutionConfigSave(t, `{"api_providers":[{"api_key":"syntax-secret"}],`)
		if !strings.Contains(recorder.Body.String(), "参数错误:") {
			t.Fatalf("expected parameter error, got %s", recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "syntax-secret") {
			t.Fatalf("invalid JSON response exposed secret: %s", recorder.Body.String())
		}
	})

	t.Run("empty body", func(t *testing.T) {
		useExecutionConfigAPITestDB(t, openExecutionConfigAPITestDB(t))
		recorder := invokeExecutionConfigSave(t, "")
		if !strings.Contains(recorder.Body.String(), "参数错误:") {
			t.Fatalf("expected empty body parameter error, got %s", recorder.Body.String())
		}
	})

	t.Run("valid prefix followed by another JSON value", func(t *testing.T) {
		db := openExecutionConfigAPITestDB(t)
		useExecutionConfigAPITestDB(t, db)
		body := `{
			"active_target":{"type":"api","id":"openai"},
			"api_providers":[{"id":"openai","label":"OpenAI","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"trailing-secret","model":"gpt-test","max_tokens":2048}],
			"cli_providers":[]
		} {"unexpected":true}`
		recorder := invokeExecutionConfigSave(t, body)
		if !strings.Contains(recorder.Body.String(), "参数错误:") {
			t.Fatalf("expected trailing JSON value to fail binding, got %s", recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "trailing-secret") {
			t.Fatalf("trailing-value response exposed secret: %s", recorder.Body.String())
		}
		var count int64
		if err := db.Model(&sysModel.AIProviderConfig{}).Count(&count).Error; err != nil {
			t.Fatalf("count API providers: %v", err)
		}
		if count != 0 {
			t.Fatalf("invalid trailing-value request persisted %d provider(s)", count)
		}
	})

	t.Run("valid value followed by trailing junk", func(t *testing.T) {
		db := openExecutionConfigAPITestDB(t)
		useExecutionConfigAPITestDB(t, db)
		recorder := invokeExecutionConfigSave(t, validExecutionConfigRequestJSON("junk-secret")+` trailing-junk`)
		if !strings.Contains(recorder.Body.String(), "参数错误:") {
			t.Fatalf("expected trailing junk to fail binding, got %s", recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "junk-secret") {
			t.Fatalf("trailing-junk response exposed secret: %s", recorder.Body.String())
		}
		var count int64
		if err := db.Model(&sysModel.AIProviderConfig{}).Count(&count).Error; err != nil {
			t.Fatalf("count API providers: %v", err)
		}
		if count != 0 {
			t.Fatalf("trailing-junk request persisted %d provider(s)", count)
		}
	})

	for _, test := range []struct {
		name string
		body string
	}{
		{name: "unknown top-level field", body: strings.TrimSuffix(validExecutionConfigRequestJSON("unknown-secret"), "\n\t}") + `,"unknown_top":true}`},
		{name: "unknown nested field", body: strings.Replace(validExecutionConfigRequestJSON("unknown-secret"), `"max_tokens":2048`, `"max_tokens":2048,"unknown_nested":true`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openExecutionConfigAPITestDB(t)
			useExecutionConfigAPITestDB(t, db)
			recorder := invokeExecutionConfigSave(t, test.body)
			if !strings.Contains(recorder.Body.String(), "参数错误:") {
				t.Fatalf("expected unknown field parameter error, got %s", recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "unknown-secret") {
				t.Fatalf("unknown-field response exposed secret: %s", recorder.Body.String())
			}
			var count int64
			if err := db.Model(&sysModel.AIProviderConfig{}).Count(&count).Error; err != nil {
				t.Fatalf("count API providers: %v", err)
			}
			if count != 0 {
				t.Fatalf("unknown-field request persisted %d provider(s)", count)
			}
		})
	}

	for _, contentType := range []string{"", "text/plain", "application/xml"} {
		name := contentType
		if name == "" {
			name = "missing"
		}
		t.Run("content type "+name, func(t *testing.T) {
			db := openExecutionConfigAPITestDB(t)
			useExecutionConfigAPITestDB(t, db)
			recorder := invokeExecutionConfigSaveWithContentType(t, validExecutionConfigRequestJSON("content-type-secret"), contentType)
			if !strings.Contains(recorder.Body.String(), "参数错误:") {
				t.Fatalf("expected content-type parameter error, got %s", recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "content-type-secret") {
				t.Fatalf("content-type response exposed secret: %s", recorder.Body.String())
			}
			var count int64
			if err := db.Model(&sysModel.AIProviderConfig{}).Count(&count).Error; err != nil {
				t.Fatalf("count API providers: %v", err)
			}
			if count != 0 {
				t.Fatalf("wrong content type persisted %d provider(s)", count)
			}
		})
	}

	t.Run("save without database", func(t *testing.T) {
		useExecutionConfigAPITestDB(t, nil)
		body := `{"active_target":{"type":"api","id":"openai"},"api_providers":[{"id":"openai","api_key":"db-failure-secret"}],"cli_providers":[]}`
		recorder := invokeExecutionConfigSave(t, body)
		if !strings.Contains(recorder.Body.String(), "保存造句执行器配置失败") {
			t.Fatalf("expected save failure, got %s", recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), "db-failure-secret") {
			t.Fatalf("save failure exposed secret: %s", recorder.Body.String())
		}
	})

	t.Run("load without database", func(t *testing.T) {
		useExecutionConfigAPITestDB(t, nil)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/ai/execution-config", nil)
		new(ExecutionConfigApi).GetConfig(ctx)
		if !strings.Contains(recorder.Body.String(), "读取造句执行器配置失败") {
			t.Fatalf("expected load failure, got %s", recorder.Body.String())
		}
	})
}

func TestExecutionConfigAPIAcceptsJSONCharsetAndTrailingWhitespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openExecutionConfigAPITestDB(t)
	useExecutionConfigAPITestDB(t, db)
	recorder := invokeExecutionConfigSaveWithContentType(
		t,
		validExecutionConfigRequestJSON("charset-secret")+" \n\t ",
		"application/json; charset=utf-8",
	)
	envelope, config := decodeExecutionConfigResponse(t, recorder)
	if envelope.Code != response.SUCCESS {
		t.Fatalf("valid JSON charset/trailing whitespace failed: %#v; body=%s", envelope, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "charset-secret") || !config.APIProviders[0].APIKeyConfigured {
		t.Fatalf("valid response exposed or misreported key: %s", recorder.Body.String())
	}
}

func TestExecutionConfigAPIReturnsSafeValidationMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		body     string
		wantText string
	}{
		{
			name:     "unknown target",
			body:     strings.Replace(validExecutionConfigRequestJSON("validation-secret"), `"id":"openai"`, `"id":"missing"`, 1),
			wantText: "API 造句执行器「missing」不存在",
		},
		{
			name:     "bad origin",
			body:     strings.Replace(validExecutionConfigRequestJSON("validation-secret"), "https://api.openai.com/v1", "https://user@example.com/v1", 1),
			wantText: "base_url 无效",
		},
		{
			name: "disabled CLI target",
			body: `{
				"active_target":{"type":"cli","id":"codex-local"},
				"api_providers":[{"id":"openai","label":"OpenAI","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"validation-secret","model":"gpt-test","max_tokens":2048}],
				"cli_providers":[{"id":"codex-local","label":"Codex Local","driver":"codex","command_path":"/usr/local/bin/codex","model":"gpt-5.6-sol","reasoning_effort":"high","working_directory":"/tmp/workspace","timeout_seconds":300,"enabled":false}]
			}`,
			wantText: "CLI 造句执行器「codex-local」已停用",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useExecutionConfigAPITestDB(t, openExecutionConfigAPITestDB(t))
			recorder := invokeExecutionConfigSave(t, test.body)
			if !strings.Contains(recorder.Body.String(), test.wantText) {
				t.Fatalf("expected safe validation message %q, got %s", test.wantText, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "validation-secret") {
				t.Fatalf("validation response exposed secret: %s", recorder.Body.String())
			}
		})
	}
}

func TestExecutionConfigAPIInternalDatabaseFailureStaysGeneric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openExecutionConfigAPITestDB(t)
	const databaseSentinel = "forced-db-secret-sentinel"
	if err := db.Callback().Create().Before("gorm:create").Register("test:execution-config-write-error", func(tx *gorm.DB) {
		tx.AddError(errors.New(databaseSentinel))
	}); err != nil {
		t.Fatalf("register forced write error: %v", err)
	}
	useExecutionConfigAPITestDB(t, db)
	recorder := invokeExecutionConfigSave(t, validExecutionConfigRequestJSON("request-secret-sentinel"))
	body := recorder.Body.String()
	if !strings.Contains(body, "保存造句执行器配置失败") {
		t.Fatalf("expected generic database failure, got %s", body)
	}
	for _, secret := range []string{databaseSentinel, "request-secret-sentinel"} {
		if strings.Contains(body, secret) {
			t.Fatalf("database failure exposed %q: %s", secret, body)
		}
	}
}
