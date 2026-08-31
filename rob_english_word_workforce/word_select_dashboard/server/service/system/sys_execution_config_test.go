package system

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func openExecutionConfigTestDB(t *testing.T) *gorm.DB {
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

func validExecutionConfigInput() ExecutionConfigInput {
	return ExecutionConfigInput{
		ActiveTarget: ExecutionTarget{Type: "api", ID: "openai"},
		APIProviders: []AIProviderInput{{
			ID:        "openai",
			Label:     " OpenAI ",
			Type:      " openai-compatible ",
			BaseURL:   " https://api.openai.com/v1/ ",
			APIKey:    "new-secret",
			Model:     " gpt-5.4-mini ",
			MaxTokens: 0,
		}},
		CLIProviders: []CLIProviderInput{{
			ID:               "codex-local",
			Label:            " Codex Local ",
			Driver:           "codex",
			CommandPath:      "/usr/local/bin/codex",
			Model:            "gpt-5.6-sol",
			ReasoningEffort:  "high",
			WorkingDirectory: "/tmp/workspace",
			TimeoutSeconds:   300,
			Enabled:          true,
		}},
	}
}

func boolValue(value bool) *bool {
	return &value
}

func TestExecutionConfigInputValidationErrorsAreTyped(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ExecutionConfigInput)
		stored map[string]storedExecutionAIProvider
	}{
		{name: "unknown target type", mutate: func(input *ExecutionConfigInput) { input.ActiveTarget.Type = "worker" }},
		{name: "blank target ID", mutate: func(input *ExecutionConfigInput) { input.ActiveTarget.ID = "" }},
		{name: "missing API target", mutate: func(input *ExecutionConfigInput) { input.ActiveTarget.ID = "missing" }},
		{name: "missing CLI target", mutate: func(input *ExecutionConfigInput) {
			input.ActiveTarget = ExecutionTarget{Type: "cli", ID: "missing"}
		}},
		{name: "disabled CLI target", mutate: func(input *ExecutionConfigInput) {
			input.ActiveTarget = ExecutionTarget{Type: "cli", ID: "codex-local"}
			input.CLIProviders[0].Enabled = false
		}},
		{name: "blank API ID", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].ID = "" }},
		{name: "duplicate API ID", mutate: func(input *ExecutionConfigInput) {
			input.APIProviders = append(input.APIProviders, input.APIProviders[0])
		}},
		{name: "blank API label", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].Label = "" }},
		{name: "blank API type", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].Type = "" }},
		{name: "blank API URL", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].BaseURL = "" }},
		{name: "invalid API origin", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].BaseURL = "https://user@example.com/v1" }},
		{name: "blank API model", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].Model = "" }},
		{name: "missing API key", mutate: func(input *ExecutionConfigInput) { input.APIProviders[0].APIKey = "" }},
		{
			name: "changed API origin with blank key",
			mutate: func(input *ExecutionConfigInput) {
				input.APIProviders[0].APIKey = ""
				input.APIProviders[0].BaseURL = "https://attacker.example/v1"
			},
			stored: map[string]storedExecutionAIProvider{
				"openai": {apiKey: "stored-secret", baseURL: "https://api.openai.com/v1"},
			},
		},
		{name: "blank CLI ID", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].ID = "" }},
		{name: "duplicate CLI ID", mutate: func(input *ExecutionConfigInput) {
			input.CLIProviders = append(input.CLIProviders, input.CLIProviders[0])
		}},
		{name: "blank CLI label", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].Label = "" }},
		{name: "invalid CLI driver", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].Driver = "claude" }},
		{name: "relative CLI command", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].CommandPath = "bin/codex" }},
		{name: "relative CLI working directory", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].WorkingDirectory = "workspace" }},
		{name: "invalid CLI timeout", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].TimeoutSeconds = 0 }},
		{name: "invalid Codex model", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].Model = "unknown" }},
		{name: "invalid Codex reasoning", mutate: func(input *ExecutionConfigInput) { input.CLIProviders[0].ReasoningEffort = "extreme" }},
		{name: "invalid Gemini model", mutate: func(input *ExecutionConfigInput) {
			input.CLIProviders[0].Driver = "gemini"
			input.CLIProviders[0].Model = "ultra"
			input.CLIProviders[0].ReasoningEffort = ""
		}},
		{name: "non-empty Gemini reasoning", mutate: func(input *ExecutionConfigInput) {
			input.CLIProviders[0].Driver = "gemini"
			input.CLIProviders[0].Model = "pro"
			input.CLIProviders[0].ReasoningEffort = "high"
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validExecutionConfigInput()
			test.mutate(&input)
			_, _, _, err := normalizeExecutionConfig(input, test.stored)
			if err == nil {
				t.Fatal("expected validation error")
			}
			var validationErr *ExecutionConfigValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected typed validation error, got %T: %v", err, err)
			}
			if !IsExecutionConfigValidationError(err) {
				t.Fatalf("validation helper rejected %T: %v", err, err)
			}
		})
	}
}

func TestExecutionConfigRejectsUnknownAndDisabledActiveTargets(t *testing.T) {
	service := new(ExecutionConfigService)

	t.Run("unknown type", func(t *testing.T) {
		input := validExecutionConfigInput()
		input.ActiveTarget.Type = "worker"
		err := service.Save(openExecutionConfigTestDB(t), input)
		if err == nil || !strings.Contains(err.Error(), "不存在") {
			t.Fatalf("expected nonexistent target error, got %v", err)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		input := validExecutionConfigInput()
		input.ActiveTarget.ID = "missing"
		err := service.Save(openExecutionConfigTestDB(t), input)
		if err == nil || !strings.Contains(err.Error(), "不存在") {
			t.Fatalf("expected nonexistent target error, got %v", err)
		}
	})

	t.Run("disabled cli", func(t *testing.T) {
		input := validExecutionConfigInput()
		input.ActiveTarget = ExecutionTarget{Type: "cli", ID: "codex-local"}
		input.CLIProviders[0].Enabled = false
		err := service.Save(openExecutionConfigTestDB(t), input)
		if err == nil || !strings.Contains(err.Error(), "已停用") {
			t.Fatalf("expected disabled target error, got %v", err)
		}
	})
}

func TestExecutionConfigRejectsDuplicateProviderIDs(t *testing.T) {
	service := new(ExecutionConfigService)

	t.Run("api", func(t *testing.T) {
		input := validExecutionConfigInput()
		input.APIProviders = append(input.APIProviders, input.APIProviders[0])
		err := service.Save(openExecutionConfigTestDB(t), input)
		if err == nil || !strings.Contains(err.Error(), "重复") {
			t.Fatalf("expected duplicate API ID error, got %v", err)
		}
	})

	t.Run("cli", func(t *testing.T) {
		input := validExecutionConfigInput()
		input.CLIProviders = append(input.CLIProviders, input.CLIProviders[0])
		err := service.Save(openExecutionConfigTestDB(t), input)
		if err == nil || !strings.Contains(err.Error(), "重复") {
			t.Fatalf("expected duplicate CLI ID error, got %v", err)
		}
	})
}

func TestExecutionConfigRejectsInvalidCLIFields(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*CLIProviderInput)
		wantError string
	}{
		{name: "driver", mutate: func(provider *CLIProviderInput) { provider.Driver = "claude" }, wantError: "driver"},
		{name: "codex model", mutate: func(provider *CLIProviderInput) { provider.Model = "gpt-unknown" }, wantError: "模型"},
		{name: "gemini model", mutate: func(provider *CLIProviderInput) {
			provider.Driver = "gemini"
			provider.Model = "ultra"
			provider.ReasoningEffort = ""
		}, wantError: "模型"},
		{name: "codex reasoning", mutate: func(provider *CLIProviderInput) { provider.ReasoningEffort = "extreme" }, wantError: "reasoning_effort"},
		{name: "gemini reasoning", mutate: func(provider *CLIProviderInput) { provider.Driver = "gemini"; provider.Model = "pro" }, wantError: "reasoning_effort"},
		{name: "relative command", mutate: func(provider *CLIProviderInput) { provider.CommandPath = "bin/codex" }, wantError: "绝对路径"},
		{name: "empty command", mutate: func(provider *CLIProviderInput) { provider.CommandPath = "" }, wantError: "绝对路径"},
		{name: "relative working directory", mutate: func(provider *CLIProviderInput) { provider.WorkingDirectory = "workspace" }, wantError: "绝对路径"},
		{name: "timeout", mutate: func(provider *CLIProviderInput) { provider.TimeoutSeconds = 0 }, wantError: "timeout_seconds"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validExecutionConfigInput()
			test.mutate(&input.CLIProviders[0])
			err := new(ExecutionConfigService).Save(openExecutionConfigTestDB(t), input)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected error containing %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestExecutionConfigPreservesKeyOnlyForSameIDAndOrigin(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	seed := sysModel.AIProviderConfig{
		ProviderID: "openai",
		Label:      "Old OpenAI",
		Type:       "openai-compatible",
		BaseURL:    "https://api.openai.com/v1",
		ApiKey:     "stored-secret",
		Model:      "old-model",
		MaxTokens:  2048,
		Active:     true,
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed API provider: %v", err)
	}

	input := validExecutionConfigInput()
	input.APIProviders[0].BaseURL = "HTTPS://API.OPENAI.COM/v2/"
	input.APIProviders[0].APIKey = ""
	if err := new(ExecutionConfigService).Save(db, input); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var persisted sysModel.AIProviderConfig
	if err := db.First(&persisted, "provider_id = ?", "openai").Error; err != nil {
		t.Fatalf("reload API provider: %v", err)
	}
	if persisted.ApiKey != "stored-secret" {
		t.Fatalf("expected stored key, got %q", persisted.ApiKey)
	}
	if persisted.BaseURL != "HTTPS://API.OPENAI.COM/v2" || persisted.MaxTokens != 4096 {
		t.Fatalf("expected normalized fields, got %#v", persisted)
	}
}

func TestExecutionAIOriginLowercasesOnlyASCIIHostnameBytes(t *testing.T) {
	tests := map[string]string{
		"https://API.EXAMPLE.COM/v1":        "https://api.example.com:443",
		"https://İ.EXAMPLE/v1":              "https://İ.example:443",
		"https://\U00001C89.EXAMPLE/v1":     "https://\U00001C89.example:443",
		"https://\U00001C8A.EXAMPLE/v1":     "https://\U00001C8A.example:443",
		"https://BÜCHER.EXAMPLE/v1":         "https://bÜcher.example:443",
		"https://[fe80::A%25ZONE]:00443/v1": "https://fe80::a%zone:00443",
	}
	for raw, want := range tests {
		t.Run(raw, func(t *testing.T) {
			got, err := normalizedExecutionAIOrigin(raw)
			if err != nil {
				t.Fatalf("normalize origin: %v", err)
			}
			if got != want {
				t.Fatalf("origin mismatch: got %q want %q", got, want)
			}
		})
	}

	upperNew, err := normalizedExecutionAIOrigin("https://\U00001C89.example/v1")
	if err != nil {
		t.Fatalf("normalize U+1C89: %v", err)
	}
	lowerNew, err := normalizedExecutionAIOrigin("https://\U00001C8A.example/v1")
	if err != nil {
		t.Fatalf("normalize U+1C8A: %v", err)
	}
	if upperNew == lowerNew {
		t.Fatalf("non-ASCII code points must remain distinct: %q", upperNew)
	}
}

func TestExecutionConfigUnicodeHostnameRequiresExactMatchForBlankKeyReuse(t *testing.T) {
	t.Run("exact Unicode hostname preserves key", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		if err := db.Create(&sysModel.AIProviderConfig{
			ProviderID: "openai", Label: "OpenAI", Type: "openai-compatible",
			BaseURL: "https://İ.example/v1", ApiKey: "stored-secret", Model: "old",
		}).Error; err != nil {
			t.Fatalf("seed provider: %v", err)
		}
		input := validExecutionConfigInput()
		input.APIProviders[0].BaseURL = "https://İ.example/v2"
		input.APIProviders[0].APIKey = ""
		if err := new(ExecutionConfigService).Save(db, input); err != nil {
			t.Fatalf("exact Unicode origin should preserve key: %v", err)
		}
	})

	t.Run("Unicode hostname change requires key", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		if err := db.Create(&sysModel.AIProviderConfig{
			ProviderID: "openai", Label: "OpenAI", Type: "openai-compatible",
			BaseURL: "https://İ.example/v1", ApiKey: "stored-secret", Model: "old",
		}).Error; err != nil {
			t.Fatalf("seed provider: %v", err)
		}
		input := validExecutionConfigInput()
		input.APIProviders[0].BaseURL = "https://i.example/v2"
		input.APIProviders[0].APIKey = ""
		err := new(ExecutionConfigService).Save(db, input)
		if err == nil || !strings.Contains(err.Error(), "服务来源已变更") {
			t.Fatalf("expected Unicode hostname change to require key, got %v", err)
		}
	})
}

func TestExecutionConfigRejectsEmptyKeyForChangedOriginOrNewID(t *testing.T) {
	service := new(ExecutionConfigService)

	t.Run("changed origin", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		if err := db.Create(&sysModel.AIProviderConfig{
			ProviderID: "openai", Label: "OpenAI", Type: "openai-compatible",
			BaseURL: "https://api.openai.com/v1", ApiKey: "stored-secret", Model: "old",
		}).Error; err != nil {
			t.Fatalf("seed provider: %v", err)
		}
		input := validExecutionConfigInput()
		input.APIProviders[0].BaseURL = "https://attacker.example/v1"
		input.APIProviders[0].APIKey = ""
		err := service.Save(db, input)
		if err == nil || !strings.Contains(err.Error(), "服务来源已变更") {
			t.Fatalf("expected changed-origin error, got %v", err)
		}
	})

	t.Run("new id", func(t *testing.T) {
		input := validExecutionConfigInput()
		input.ActiveTarget.ID = "new-openai"
		input.APIProviders[0].ID = "new-openai"
		input.APIProviders[0].APIKey = ""
		err := service.Save(openExecutionConfigTestDB(t), input)
		if err == nil || !strings.Contains(err.Error(), "API Key") {
			t.Fatalf("expected missing-key error, got %v", err)
		}
	})
}

func TestExecutionConfigSaveSynchronizesProvidersLegacyActiveAndSingleton(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	oldAPIs := []sysModel.AIProviderConfig{
		{ProviderID: "openai", Label: "Old", Type: "openai-compatible", BaseURL: "https://api.openai.com/v1", ApiKey: "old-secret", Model: "old", Active: false},
		{ProviderID: "stale-api", Label: "Stale", Type: "openai-compatible", BaseURL: "https://stale.example/v1", ApiKey: "stale", Model: "stale", Active: true},
	}
	oldCLIs := []sysModel.CLIProviderConfig{
		{ProviderID: "codex-local", Label: "Old", Driver: "codex", CommandPath: "/old/codex", Model: "gpt-5.4", ReasoningEffort: "low", WorkingDirectory: "/old", TimeoutSeconds: 30, Enabled: boolValue(false)},
		{ProviderID: "stale-cli", Label: "Stale", Driver: "gemini", CommandPath: "/old/gemini", Model: "auto", WorkingDirectory: "/old", TimeoutSeconds: 30, Enabled: boolValue(true)},
	}
	if err := db.Create(&oldAPIs).Error; err != nil {
		t.Fatalf("seed APIs: %v", err)
	}
	if err := db.Create(&oldCLIs).Error; err != nil {
		t.Fatalf("seed CLIs: %v", err)
	}

	input := validExecutionConfigInput()
	input.APIProviders = append(input.APIProviders, AIProviderInput{
		ID: "anthropic", Label: "Anthropic", Type: "anthropic-compatible",
		BaseURL: "https://api.anthropic.com/v1", APIKey: "anthropic-secret", Model: "claude",
	})
	input.ActiveTarget = ExecutionTarget{Type: "api", ID: "anthropic"}
	if err := new(ExecutionConfigService).Save(db, input); err != nil {
		t.Fatalf("save API target config: %v", err)
	}

	var apis []sysModel.AIProviderConfig
	if err := db.Order("provider_id").Find(&apis).Error; err != nil {
		t.Fatalf("load APIs: %v", err)
	}
	if len(apis) != 2 || apis[0].ProviderID != "anthropic" || apis[1].ProviderID != "openai" {
		t.Fatalf("unexpected API rows: %#v", apis)
	}
	if !apis[0].Active || apis[1].Active {
		t.Fatalf("expected only anthropic active, got %#v", apis)
	}

	var clis []sysModel.CLIProviderConfig
	if err := db.Order("provider_id").Find(&clis).Error; err != nil {
		t.Fatalf("load CLIs: %v", err)
	}
	if len(clis) != 1 || clis[0].ProviderID != "codex-local" || clis[0].CommandPath != "/usr/local/bin/codex" {
		t.Fatalf("unexpected CLI rows: %#v", clis)
	}

	var target sysModel.SentenceExecutorConfig
	if err := db.First(&target, "singleton_key = ?", "default").Error; err != nil {
		t.Fatalf("load target: %v", err)
	}
	if target.SingletonKey != "default" || target.ExecutorType != "api" || target.ExecutorID != "anthropic" {
		t.Fatalf("unexpected target: %#v", target)
	}

	input.ActiveTarget = ExecutionTarget{Type: "cli", ID: "codex-local"}
	if err := new(ExecutionConfigService).Save(db, input); err != nil {
		t.Fatalf("save CLI target config: %v", err)
	}
	var activeAPIs int64
	if err := db.Model(&sysModel.AIProviderConfig{}).Where("active = ?", true).Count(&activeAPIs).Error; err != nil {
		t.Fatalf("count active APIs: %v", err)
	}
	if activeAPIs != 0 {
		t.Fatalf("expected all legacy API active flags cleared, got %d", activeAPIs)
	}
}

func TestExecutionConfigSaveSafelyDeletesEmptyProviderLists(t *testing.T) {
	service := new(ExecutionConfigService)

	t.Run("empty api list", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		if err := db.Create(&sysModel.AIProviderConfig{ProviderID: "stale", Label: "Stale", Type: "openai-compatible", BaseURL: "https://stale.example", ApiKey: "secret", Model: "old"}).Error; err != nil {
			t.Fatalf("seed API: %v", err)
		}
		input := validExecutionConfigInput()
		input.APIProviders = nil
		input.ActiveTarget = ExecutionTarget{Type: "cli", ID: "codex-local"}
		if err := service.Save(db, input); err != nil {
			t.Fatalf("save empty APIs: %v", err)
		}
		var count int64
		if err := db.Model(&sysModel.AIProviderConfig{}).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("expected no APIs, count=%d err=%v", count, err)
		}
	})

	t.Run("empty cli list", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		if err := db.Create(&sysModel.CLIProviderConfig{ProviderID: "stale", Label: "Stale", Driver: "codex", CommandPath: "/bin/codex", Model: "gpt-5.4", WorkingDirectory: "/tmp", TimeoutSeconds: 30, Enabled: boolValue(true)}).Error; err != nil {
			t.Fatalf("seed CLI: %v", err)
		}
		input := validExecutionConfigInput()
		input.CLIProviders = nil
		if err := service.Save(db, input); err != nil {
			t.Fatalf("save empty CLIs: %v", err)
		}
		var count int64
		if err := db.Model(&sysModel.CLIProviderConfig{}).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("expected no CLIs, count=%d err=%v", count, err)
		}
	})
}

func TestExecutionConfigSaveRollsBackAllRowsWhenTargetWriteFails(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	originalAPI := sysModel.AIProviderConfig{ProviderID: "old-api", Label: "Old API", Type: "openai-compatible", BaseURL: "https://old.example/v1", ApiKey: "old-secret", Model: "old-model", Active: true}
	originalCLI := sysModel.CLIProviderConfig{ProviderID: "old-cli", Label: "Old CLI", Driver: "codex", CommandPath: "/old/codex", Model: "gpt-5.4", ReasoningEffort: "low", WorkingDirectory: "/old", TimeoutSeconds: 30, Enabled: boolValue(true)}
	originalTarget := sysModel.SentenceExecutorConfig{SingletonKey: "default", ExecutorType: "api", ExecutorID: "old-api"}
	if err := db.Create(&originalAPI).Error; err != nil {
		t.Fatalf("seed API: %v", err)
	}
	if err := db.Create(&originalCLI).Error; err != nil {
		t.Fatalf("seed CLI: %v", err)
	}
	if err := db.Create(&originalTarget).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	callbackName := "execution_config_test:fail_target_write"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == originalTarget.TableName() {
			tx.AddError(errors.New("forced target failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove(callbackName) })

	input := validExecutionConfigInput()
	input.ActiveTarget = ExecutionTarget{Type: "cli", ID: "codex-local"}
	err := new(ExecutionConfigService).Save(db, input)
	if err == nil || !strings.Contains(err.Error(), "forced target failure") {
		t.Fatalf("expected target failure, got %v", err)
	}

	var apis []sysModel.AIProviderConfig
	var clis []sysModel.CLIProviderConfig
	var target sysModel.SentenceExecutorConfig
	if err := db.Find(&apis).Error; err != nil {
		t.Fatalf("reload APIs: %v", err)
	}
	if err := db.Find(&clis).Error; err != nil {
		t.Fatalf("reload CLIs: %v", err)
	}
	if err := db.First(&target, "singleton_key = ?", "default").Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if len(apis) != 1 || apis[0].ProviderID != originalAPI.ProviderID || !apis[0].Active {
		t.Fatalf("API rows were not rolled back: %#v", apis)
	}
	if len(clis) != 1 || clis[0].ProviderID != originalCLI.ProviderID {
		t.Fatalf("CLI rows were not rolled back: %#v", clis)
	}
	if target.ExecutorType != originalTarget.ExecutorType || target.ExecutorID != originalTarget.ExecutorID {
		t.Fatalf("target was not rolled back: %#v", target)
	}
}

func TestExecutionConfigLoadMigratesOnlyUniqueLegacyActiveAPI(t *testing.T) {
	t.Run("unique active", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		rows := []sysModel.AIProviderConfig{
			{ProviderID: "inactive", Label: "Inactive", Type: "openai-compatible", BaseURL: "https://inactive.example/v1", ApiKey: "inactive-secret", Model: "inactive", Active: false},
			{ProviderID: "active", Label: "Active", Type: "openai-compatible", BaseURL: "https://active.example/v1", ApiKey: "active-secret", Model: "active-model", Active: true},
		}
		if err := db.Create(&rows).Error; err != nil {
			t.Fatalf("seed APIs: %v", err)
		}
		loaded, err := new(ExecutionConfigService).Load(db)
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if loaded.ActiveTarget != (ExecutionTarget{Type: "api", ID: "active"}) {
			t.Fatalf("unexpected migrated target: %#v", loaded.ActiveTarget)
		}
		if len(loaded.APIProviders) != 2 || loaded.APIProviders[0].APIKey == "" || loaded.APIProviders[1].APIKey == "" {
			t.Fatalf("expected raw API keys in service result: %#v", loaded.APIProviders)
		}
		var target sysModel.SentenceExecutorConfig
		if err := db.First(&target, "singleton_key = ?", "default").Error; err != nil {
			t.Fatalf("expected persisted migrated target: %v", err)
		}
		if target.ExecutorID != "active" {
			t.Fatalf("unexpected persisted target: %#v", target)
		}
	})

	t.Run("no active does not fall back alphabetically", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		rows := []sysModel.AIProviderConfig{
			{ProviderID: "alpha", Label: "Alpha", Type: "openai-compatible", BaseURL: "https://alpha.example/v1", ApiKey: "alpha", Model: "alpha"},
			{ProviderID: "beta", Label: "Beta", Type: "openai-compatible", BaseURL: "https://beta.example/v1", ApiKey: "beta", Model: "beta"},
		}
		if err := db.Create(&rows).Error; err != nil {
			t.Fatalf("seed APIs: %v", err)
		}
		_, err := new(ExecutionConfigService).Load(db)
		if err == nil || !strings.Contains(err.Error(), "尚未选择造句执行器") {
			t.Fatalf("expected no-executor error, got %v", err)
		}
	})

	t.Run("multiple active is ambiguous", func(t *testing.T) {
		db := openExecutionConfigTestDB(t)
		rows := []sysModel.AIProviderConfig{
			{ProviderID: "alpha", Label: "Alpha", Type: "openai-compatible", BaseURL: "https://alpha.example/v1", ApiKey: "alpha", Model: "alpha", Active: true},
			{ProviderID: "beta", Label: "Beta", Type: "openai-compatible", BaseURL: "https://beta.example/v1", ApiKey: "beta", Model: "beta", Active: true},
		}
		if err := db.Create(&rows).Error; err != nil {
			t.Fatalf("seed APIs: %v", err)
		}
		_, err := new(ExecutionConfigService).Load(db)
		if err == nil || !strings.Contains(err.Error(), "多个") {
			t.Fatalf("expected ambiguous-active error, got %v", err)
		}
	})
}

func TestExecutionConfigDisabledCLIStaysFalseAcrossSaveLoadAndDatabase(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	input := validExecutionConfigInput()
	input.CLIProviders[0].Enabled = false
	if err := new(ExecutionConfigService).Save(db, input); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := new(ExecutionConfigService).Load(db)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(loaded.CLIProviders) != 1 || loaded.CLIProviders[0].Enabled {
		t.Fatalf("expected loaded CLI disabled, got %#v", loaded.CLIProviders)
	}
	var persisted sysModel.CLIProviderConfig
	if err := db.First(&persisted, "provider_id = ?", "codex-local").Error; err != nil {
		t.Fatalf("load persisted CLI: %v", err)
	}
	if persisted.Enabled == nil || *persisted.Enabled {
		t.Fatalf("expected explicit persisted false, got %#v", persisted.Enabled)
	}
}

func TestExecutionConfigSaveAndLoadReturnsNormalizedConfiguration(t *testing.T) {
	loaded, err := new(ExecutionConfigService).SaveAndLoad(openExecutionConfigTestDB(t), validExecutionConfigInput())
	if err != nil {
		t.Fatalf("save and load: %v", err)
	}
	if loaded.ActiveTarget != (ExecutionTarget{Type: "api", ID: "openai"}) {
		t.Fatalf("unexpected target: %#v", loaded.ActiveTarget)
	}
	if len(loaded.APIProviders) != 1 || loaded.APIProviders[0].BaseURL != "https://api.openai.com/v1" || loaded.APIProviders[0].MaxTokens != 4096 {
		t.Fatalf("unexpected API providers: %#v", loaded.APIProviders)
	}
	if loaded.APIProviders[0].APIKey != "new-secret" {
		t.Fatalf("expected raw service-layer API key, got %q", loaded.APIProviders[0].APIKey)
	}
	if len(loaded.CLIProviders) != 1 || loaded.CLIProviders[0].Label != "Codex Local" {
		t.Fatalf("unexpected CLI providers: %#v", loaded.CLIProviders)
	}
}

func TestExecutionConfigLoadFallsBackEmptyLegacyLabelsToProviderID(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	enabled := true
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "legacy-api", Type: "openai-compatible", BaseURL: "https://api.example/v1",
		ApiKey: "secret", Model: "model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed API: %v", err)
	}
	if err := db.Create(&sysModel.CLIProviderConfig{
		ProviderID: "legacy-cli", Driver: "codex", CommandPath: "/bin/codex", Model: "gpt-5.4",
		ReasoningEffort: "low", WorkingDirectory: "/tmp", TimeoutSeconds: 30, Enabled: &enabled,
	}).Error; err != nil {
		t.Fatalf("seed CLI: %v", err)
	}

	loaded, err := new(ExecutionConfigService).Load(db)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.APIProviders[0].Label != "legacy-api" || loaded.CLIProviders[0].Label != "legacy-cli" {
		t.Fatalf("legacy labels did not fall back to IDs: %#v %#v", loaded.APIProviders, loaded.CLIProviders)
	}
}

type executionConfigContextKey string

const pauseExecutionConfigLoad executionConfigContextKey = "pause-execution-config-load"

func TestExecutionConfigSaveAndLoadReturnsItsOwnCommittedInput(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	queryStarted := make(chan struct{})
	releaseQuery := make(chan struct{})
	var signalOnce sync.Once
	var saveFinished atomic.Bool
	callbackName := "execution_config_test:pause_post_save_load"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		finished, _ := tx.Statement.Context.Value(pauseExecutionConfigLoad).(*atomic.Bool)
		if finished == nil || !finished.Load() {
			return
		}
		signalOnce.Do(func() { close(queryStarted) })
		<-releaseQuery
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	createCallbackName := "execution_config_test:mark_save_finished"
	if err := db.Callback().Create().After("gorm:create").Register(createCallbackName, func(tx *gorm.DB) {
		finished, _ := tx.Statement.Context.Value(pauseExecutionConfigLoad).(*atomic.Bool)
		if finished != nil && tx.Statement.Table == (sysModel.SentenceExecutorConfig{}).TableName() {
			finished.Store(true)
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
		_ = db.Callback().Create().Remove(createCallbackName)
	})

	inputA := validExecutionConfigInput()
	inputA.ActiveTarget.ID = "api-a"
	inputA.APIProviders[0].ID = "api-a"
	inputA.APIProviders[0].APIKey = "secret-a"
	inputB := validExecutionConfigInput()
	inputB.ActiveTarget.ID = "api-b"
	inputB.APIProviders[0].ID = "api-b"
	inputB.APIProviders[0].APIKey = "secret-b"

	type saveResult struct {
		config ExecutionConfigInput
		err    error
	}
	resultA := make(chan saveResult, 1)
	dbA := db.WithContext(context.WithValue(context.Background(), pauseExecutionConfigLoad, &saveFinished))
	go func() {
		loaded, err := new(ExecutionConfigService).SaveAndLoad(dbA, inputA)
		resultA <- saveResult{config: loaded, err: err}
	}()

	select {
	case <-queryStarted:
		if err := new(ExecutionConfigService).Save(db, inputB); err != nil {
			close(releaseQuery)
			t.Fatalf("save competing config: %v", err)
		}
		close(releaseQuery)
	case early := <-resultA:
		if early.err != nil {
			t.Fatalf("save and load A: %v", early.err)
		}
		if early.config.ActiveTarget.ID != "api-a" {
			t.Fatalf("save and load returned competing target: %#v", early.config.ActiveTarget)
		}
		if err := new(ExecutionConfigService).Save(db, inputB); err != nil {
			t.Fatalf("save competing config: %v", err)
		}
		return
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SaveAndLoad")
	}

	select {
	case result := <-resultA:
		if result.err != nil {
			t.Fatalf("save and load A: %v", result.err)
		}
		if result.config.ActiveTarget.ID != "api-a" || result.config.APIProviders[0].ID != "api-a" {
			t.Fatalf("SaveAndLoad returned another save's config: %#v", result.config)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SaveAndLoad result")
	}
}

func TestExecutionConfigLoadUsesTargetFirstAndConflictSafeMigration(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "legacy", Label: "Legacy", Type: "openai-compatible",
		BaseURL: "https://legacy.example/v1", ApiKey: "secret", Model: "model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	var mu sync.Mutex
	queryTables := make([]string, 0, 4)
	lockedTargetRead := false
	conflictSafeCreate := false
	queryCallback := "execution_config_test:record_query_order"
	createCallback := "execution_config_test:record_migration_conflict_clause"
	if err := db.Callback().Query().Before("gorm:query").Register(queryCallback, func(tx *gorm.DB) {
		mu.Lock()
		defer mu.Unlock()
		queryTables = append(queryTables, tx.Statement.Table)
		if tx.Statement.Table == (sysModel.SentenceExecutorConfig{}).TableName() {
			if locking, exists := tx.Statement.Clauses["FOR"]; exists {
				if expression, ok := locking.Expression.(clause.Locking); ok && expression.Strength == "SHARE" {
					lockedTargetRead = true
				}
			}
		}
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register(createCallback, func(tx *gorm.DB) {
		if tx.Statement.Table != (sysModel.SentenceExecutorConfig{}).TableName() {
			return
		}
		if conflict, exists := tx.Statement.Clauses["ON CONFLICT"]; exists {
			if expression, ok := conflict.Expression.(clause.OnConflict); ok && expression.DoNothing {
				conflictSafeCreate = true
			}
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(queryCallback)
		_ = db.Callback().Create().Remove(createCallback)
	})

	if _, err := new(ExecutionConfigService).Load(db); err != nil {
		t.Fatalf("load config: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(queryTables) == 0 || queryTables[0] != (sysModel.SentenceExecutorConfig{}).TableName() {
		t.Fatalf("expected singleton query first, got %v", queryTables)
	}
	if !lockedTargetRead {
		t.Fatal("singleton was not read with FOR SHARE")
	}
	if !conflictSafeCreate {
		t.Fatal("legacy migration did not use ON CONFLICT DO NOTHING")
	}
}

func TestExecutionConfigConcurrentLegacyMigrationConverges(t *testing.T) {
	db := openExecutionConfigTestDB(t)
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "legacy", Label: "Legacy", Type: "openai-compatible",
		BaseURL: "https://legacy.example/v1", ApiKey: "secret", Model: "model", Active: true,
	}).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			loaded, err := new(ExecutionConfigService).Load(db)
			if err == nil && loaded.ActiveTarget != (ExecutionTarget{Type: "api", ID: "legacy"}) {
				err = errors.New("unexpected migrated target")
			}
			results <- err
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("concurrent legacy migration: %v", err)
		}
	}
	var count int64
	if err := db.Model(&sysModel.SentenceExecutorConfig{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected one singleton, count=%d err=%v", count, err)
	}
}

func TestExecutionConfigPostgresGuardStatement(t *testing.T) {
	statement, enabled := executionConfigGuardStatement("postgres")
	if !enabled || statement != "SELECT pg_advisory_xact_lock(?)" {
		t.Fatalf("unexpected PostgreSQL guard: enabled=%v statement=%q", enabled, statement)
	}
	if statement, enabled := executionConfigGuardStatement("sqlite"); enabled || statement != "" {
		t.Fatalf("unexpected SQLite advisory guard: enabled=%v statement=%q", enabled, statement)
	}
}
