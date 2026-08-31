package system

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	sysModel "github.com/conchi/go-react-template/server/model/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const executionConfigSingletonKey = "default"

const executionConfigPostgresAdvisoryLockKey int64 = 0x45584543434647

var (
	executionConfigWriteCoordinator sync.Mutex
	executionConfigCoordinatorHook  func()

	codexModels = map[string]struct{}{
		"gpt-5.6-sol": {}, "gpt-5.6-terra": {}, "gpt-5.6-luna": {},
		"gpt-5.5": {}, "gpt-5.4": {}, "gpt-5.4-mini": {}, "gpt-5.3-codex-spark": {},
	}
	geminiModels = map[string]struct{}{
		"auto": {}, "pro": {}, "flash": {}, "flash-lite": {},
	}
	codexReasoningEfforts = map[string]struct{}{
		"low": {}, "medium": {}, "high": {}, "xhigh": {},
	}
)

type ExecutionTarget struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AIProviderInput struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
}

type CLIProviderInput struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Driver           string `json:"driver"`
	CommandPath      string `json:"command_path"`
	Model            string `json:"model"`
	ReasoningEffort  string `json:"reasoning_effort"`
	WorkingDirectory string `json:"working_directory"`
	TimeoutSeconds   int    `json:"timeout_seconds"`
	Enabled          bool   `json:"enabled"`
}

type ExecutionConfigInput struct {
	ActiveTarget ExecutionTarget    `json:"active_target"`
	APIProviders []AIProviderInput  `json:"api_providers"`
	CLIProviders []CLIProviderInput `json:"cli_providers"`
}

type ExecutionConfigService struct{}

type ExecutionConfigValidationError struct {
	message string
}

func (e *ExecutionConfigValidationError) Error() string {
	return e.message
}

func IsExecutionConfigValidationError(err error) bool {
	var validationErr *ExecutionConfigValidationError
	return errors.As(err, &validationErr)
}

func executionConfigValidationErrorf(format string, args ...interface{}) error {
	return &ExecutionConfigValidationError{message: fmt.Sprintf(format, args...)}
}

type storedExecutionAIProvider struct {
	apiKey  string
	baseURL string
}

func (s *ExecutionConfigService) Save(db *gorm.DB, input ExecutionConfigInput) error {
	if db == nil {
		return errors.New("数据库未连接，无法保存造句执行器配置")
	}

	return withExecutionConfigWriteCoordinator(func() error {
		return db.Transaction(func(tx *gorm.DB) error {
			_, err := saveExecutionConfigTransaction(tx, input)
			return err
		})
	})
}

func (s *ExecutionConfigService) Load(db *gorm.DB) (ExecutionConfigInput, error) {
	if db == nil {
		return ExecutionConfigInput{}, errors.New("数据库未连接，无法读取造句执行器配置")
	}

	var result ExecutionConfigInput
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := acquireExecutionConfigGuard(tx); err != nil {
			return err
		}
		target, err := loadOrMigrateExecutionTarget(tx)
		if err != nil {
			return err
		}

		var apiRows []sysModel.AIProviderConfig
		if err := tx.Order("provider_id ASC").Find(&apiRows).Error; err != nil {
			return err
		}
		var cliRows []sysModel.CLIProviderConfig
		if err := tx.Order("provider_id ASC").Find(&cliRows).Error; err != nil {
			return err
		}
		result = buildExecutionConfigResult(apiRows, cliRows, target)
		return validateLoadedExecutionTarget(result)
	})
	if err != nil {
		return ExecutionConfigInput{}, err
	}
	return result, nil
}

func (s *ExecutionConfigService) SaveAndLoad(db *gorm.DB, input ExecutionConfigInput) (ExecutionConfigInput, error) {
	if db == nil {
		return ExecutionConfigInput{}, errors.New("数据库未连接，无法保存造句执行器配置")
	}

	var result ExecutionConfigInput
	err := withExecutionConfigWriteCoordinator(func() error {
		return db.Transaction(func(tx *gorm.DB) error {
			var err error
			result, err = saveExecutionConfigTransaction(tx, input)
			return err
		})
	})
	if err != nil {
		return ExecutionConfigInput{}, err
	}
	return result, nil
}

func withExecutionConfigWriteCoordinator(work func() error) error {
	if executionConfigCoordinatorHook != nil {
		executionConfigCoordinatorHook()
	}
	executionConfigWriteCoordinator.Lock()
	defer executionConfigWriteCoordinator.Unlock()
	return work()
}

func saveExecutionConfigTransaction(tx *gorm.DB, input ExecutionConfigInput) (ExecutionConfigInput, error) {
	if err := acquireExecutionConfigGuard(tx); err != nil {
		return ExecutionConfigInput{}, err
	}
	if _, _, err := findExecutionTarget(tx, "UPDATE"); err != nil {
		return ExecutionConfigInput{}, err
	}

	storedProviders, err := loadStoredExecutionAIProviders(tx)
	if err != nil {
		return ExecutionConfigInput{}, err
	}
	normalized, apiRows, cliRows, err := normalizeExecutionConfig(input, storedProviders)
	if err != nil {
		return ExecutionConfigInput{}, err
	}

	if err := upsertExecutionAPIProviders(tx, apiRows); err != nil {
		return ExecutionConfigInput{}, err
	}
	if err := deleteMissingExecutionAPIProviders(tx, providerIDsFromAPIRows(apiRows)); err != nil {
		return ExecutionConfigInput{}, err
	}
	if err := upsertExecutionCLIProviders(tx, cliRows); err != nil {
		return ExecutionConfigInput{}, err
	}
	if err := deleteMissingExecutionCLIProviders(tx, providerIDsFromCLIRows(cliRows)); err != nil {
		return ExecutionConfigInput{}, err
	}

	target := sysModel.SentenceExecutorConfig{
		SingletonKey: executionConfigSingletonKey,
		ExecutorType: normalized.ActiveTarget.Type,
		ExecutorID:   normalized.ActiveTarget.ID,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "singleton_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"executor_type", "executor_id", "updated_at"}),
	}).Create(&target).Error; err != nil {
		return ExecutionConfigInput{}, err
	}
	return normalized, nil
}

func acquireExecutionConfigGuard(tx *gorm.DB) error {
	statement, enabled := executionConfigGuardStatement(tx.Dialector.Name())
	if !enabled {
		return nil
	}
	return tx.Exec(statement, executionConfigPostgresAdvisoryLockKey).Error
}

func executionConfigGuardStatement(dialect string) (string, bool) {
	if dialect == "postgres" {
		return "SELECT pg_advisory_xact_lock(?)", true
	}
	return "", false
}

func loadOrMigrateExecutionTarget(tx *gorm.DB) (sysModel.SentenceExecutorConfig, error) {
	target, found, err := findExecutionTarget(tx, "SHARE")
	if err != nil || found {
		return target, err
	}

	var activeAPIs []sysModel.AIProviderConfig
	if err := tx.Select("provider_id").Where("active = ?", true).Order("provider_id ASC").Find(&activeAPIs).Error; err != nil {
		return sysModel.SentenceExecutorConfig{}, err
	}
	if len(activeAPIs) != 1 {
		// A concurrent Save may have inserted the singleton after the first read.
		if target, found, err = findExecutionTarget(tx, "SHARE"); err != nil || found {
			return target, err
		}
		if len(activeAPIs) == 0 {
			return sysModel.SentenceExecutorConfig{}, errors.New("尚未选择造句执行器")
		}
		return sysModel.SentenceExecutorConfig{}, errors.New("存在多个旧版 active API 配置，无法确定造句执行器")
	}

	candidate := sysModel.SentenceExecutorConfig{
		SingletonKey: executionConfigSingletonKey,
		ExecutorType: "api",
		ExecutorID:   strings.TrimSpace(activeAPIs[0].ProviderID),
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate).Error; err != nil {
		return sysModel.SentenceExecutorConfig{}, err
	}
	target, found, err = findExecutionTarget(tx, "SHARE")
	if err != nil {
		return sysModel.SentenceExecutorConfig{}, err
	}
	if !found {
		return sysModel.SentenceExecutorConfig{}, errors.New("尚未选择造句执行器")
	}
	return target, nil
}

func findExecutionTarget(tx *gorm.DB, lockStrength string) (sysModel.SentenceExecutorConfig, bool, error) {
	var target sysModel.SentenceExecutorConfig
	query := tx.Where("singleton_key = ?", executionConfigSingletonKey).Limit(1)
	if lockStrength != "" {
		query = query.Clauses(clause.Locking{Strength: lockStrength})
	}
	result := query.Find(&target)
	return target, result.RowsAffected == 1, result.Error
}

func loadStoredExecutionAIProviders(tx *gorm.DB) (map[string]storedExecutionAIProvider, error) {
	var rows []sysModel.AIProviderConfig
	if err := tx.Find(&rows).Error; err != nil {
		return nil, err
	}
	providers := make(map[string]storedExecutionAIProvider, len(rows))
	for _, row := range rows {
		providers[strings.TrimSpace(row.ProviderID)] = storedExecutionAIProvider{
			apiKey:  strings.TrimSpace(row.ApiKey),
			baseURL: strings.TrimSpace(row.BaseURL),
		}
	}
	return providers, nil
}

func normalizeExecutionConfig(input ExecutionConfigInput, storedProviders map[string]storedExecutionAIProvider) (ExecutionConfigInput, []sysModel.AIProviderConfig, []sysModel.CLIProviderConfig, error) {
	target := ExecutionTarget{
		Type: strings.TrimSpace(input.ActiveTarget.Type),
		ID:   strings.TrimSpace(input.ActiveTarget.ID),
	}
	if target.Type != "api" && target.Type != "cli" {
		return ExecutionConfigInput{}, nil, nil, executionConfigValidationErrorf("造句执行器类型「%s」不存在", target.Type)
	}
	if target.ID == "" {
		return ExecutionConfigInput{}, nil, nil, executionConfigValidationErrorf("造句执行器不存在：ID 不能为空")
	}

	normalizedAPIs, apiRows, apiIDs, err := normalizeExecutionAPIProviders(input.APIProviders, storedProviders, target)
	if err != nil {
		return ExecutionConfigInput{}, nil, nil, err
	}
	normalizedCLIs, cliRows, cliIDs, err := normalizeExecutionCLIProviders(input.CLIProviders)
	if err != nil {
		return ExecutionConfigInput{}, nil, nil, err
	}

	switch target.Type {
	case "api":
		if _, exists := apiIDs[target.ID]; !exists {
			return ExecutionConfigInput{}, nil, nil, executionConfigValidationErrorf("API 造句执行器「%s」不存在", target.ID)
		}
	case "cli":
		enabled, exists := cliIDs[target.ID]
		if !exists {
			return ExecutionConfigInput{}, nil, nil, executionConfigValidationErrorf("CLI 造句执行器「%s」不存在", target.ID)
		}
		if !enabled {
			return ExecutionConfigInput{}, nil, nil, executionConfigValidationErrorf("CLI 造句执行器「%s」已停用", target.ID)
		}
	}

	return ExecutionConfigInput{
		ActiveTarget: target,
		APIProviders: normalizedAPIs,
		CLIProviders: normalizedCLIs,
	}, apiRows, cliRows, nil
}

func normalizeExecutionAPIProviders(inputs []AIProviderInput, storedProviders map[string]storedExecutionAIProvider, target ExecutionTarget) ([]AIProviderInput, []sysModel.AIProviderConfig, map[string]struct{}, error) {
	normalized := make([]AIProviderInput, 0, len(inputs))
	rows := make([]sysModel.AIProviderConfig, 0, len(inputs))
	ids := make(map[string]struct{}, len(inputs))

	for _, input := range inputs {
		provider := AIProviderInput{
			ID:        strings.TrimSpace(input.ID),
			Label:     strings.TrimSpace(input.Label),
			Type:      strings.TrimSpace(input.Type),
			BaseURL:   strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
			APIKey:    strings.TrimSpace(input.APIKey),
			Model:     strings.TrimSpace(input.Model),
			MaxTokens: input.MaxTokens,
		}
		if provider.ID == "" {
			return nil, nil, nil, executionConfigValidationErrorf("API provider ID 不能为空")
		}
		if _, exists := ids[provider.ID]; exists {
			return nil, nil, nil, executionConfigValidationErrorf("API provider ID「%s」重复", provider.ID)
		}
		ids[provider.ID] = struct{}{}
		if provider.Label == "" {
			return nil, nil, nil, executionConfigValidationErrorf("API provider「%s」的 label 不能为空", provider.ID)
		}
		if provider.Type == "" {
			return nil, nil, nil, executionConfigValidationErrorf("API provider「%s」的 type 不能为空", provider.ID)
		}
		if provider.BaseURL == "" {
			return nil, nil, nil, executionConfigValidationErrorf("API provider「%s」的 base_url 不能为空", provider.ID)
		}
		origin, err := normalizedExecutionAIOrigin(provider.BaseURL)
		if err != nil {
			return nil, nil, nil, executionConfigValidationErrorf("API provider「%s」的 base_url 无效", provider.ID)
		}
		if provider.Model == "" {
			return nil, nil, nil, executionConfigValidationErrorf("API provider「%s」的 model 不能为空", provider.ID)
		}
		if provider.MaxTokens <= 0 {
			provider.MaxTokens = 4096
		}
		if provider.APIKey == "" {
			stored, exists := storedProviders[provider.ID]
			if !exists || strings.TrimSpace(stored.apiKey) == "" {
				return nil, nil, nil, executionConfigValidationErrorf("请填写 API provider「%s」的 API Key", provider.ID)
			}
			storedOrigin, storedOriginErr := normalizedExecutionAIOrigin(stored.baseURL)
			if storedOriginErr != nil || origin != storedOrigin {
				return nil, nil, nil, executionConfigValidationErrorf("API provider「%s」的服务来源已变更，请重新填写 API Key", provider.ID)
			}
			provider.APIKey = strings.TrimSpace(stored.apiKey)
		}

		normalized = append(normalized, provider)
		rows = append(rows, sysModel.AIProviderConfig{
			ProviderID: provider.ID,
			Label:      provider.Label,
			Type:       provider.Type,
			BaseURL:    provider.BaseURL,
			ApiKey:     provider.APIKey,
			Model:      provider.Model,
			MaxTokens:  provider.MaxTokens,
			Active:     target.Type == "api" && target.ID == provider.ID,
		})
	}
	return normalized, rows, ids, nil
}

func normalizeExecutionCLIProviders(inputs []CLIProviderInput) ([]CLIProviderInput, []sysModel.CLIProviderConfig, map[string]bool, error) {
	normalized := make([]CLIProviderInput, 0, len(inputs))
	rows := make([]sysModel.CLIProviderConfig, 0, len(inputs))
	ids := make(map[string]bool, len(inputs))

	for _, input := range inputs {
		provider := CLIProviderInput{
			ID:               strings.TrimSpace(input.ID),
			Label:            strings.TrimSpace(input.Label),
			Driver:           strings.TrimSpace(input.Driver),
			CommandPath:      strings.TrimSpace(input.CommandPath),
			Model:            strings.TrimSpace(input.Model),
			ReasoningEffort:  strings.TrimSpace(input.ReasoningEffort),
			WorkingDirectory: strings.TrimSpace(input.WorkingDirectory),
			TimeoutSeconds:   input.TimeoutSeconds,
			Enabled:          input.Enabled,
		}
		if provider.ID == "" {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider ID 不能为空")
		}
		if _, exists := ids[provider.ID]; exists {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider ID「%s」重复", provider.ID)
		}
		ids[provider.ID] = provider.Enabled
		if provider.Label == "" {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider「%s」的 label 不能为空", provider.ID)
		}
		if provider.Driver != "codex" && provider.Driver != "gemini" {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider「%s」的 driver 仅支持 codex 或 gemini", provider.ID)
		}
		if provider.CommandPath == "" || !filepath.IsAbs(provider.CommandPath) {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider「%s」的 command_path 必须是非空绝对路径", provider.ID)
		}
		if provider.WorkingDirectory == "" || !filepath.IsAbs(provider.WorkingDirectory) {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider「%s」的 working_directory 必须是非空绝对路径", provider.ID)
		}
		if provider.TimeoutSeconds <= 0 {
			return nil, nil, nil, executionConfigValidationErrorf("CLI provider「%s」的 timeout_seconds 必须大于 0", provider.ID)
		}
		if err := validateExecutionCLIModelAndReasoning(provider); err != nil {
			return nil, nil, nil, err
		}

		enabled := provider.Enabled
		normalized = append(normalized, provider)
		rows = append(rows, sysModel.CLIProviderConfig{
			ProviderID:       provider.ID,
			Label:            provider.Label,
			Driver:           provider.Driver,
			CommandPath:      provider.CommandPath,
			Model:            provider.Model,
			ReasoningEffort:  provider.ReasoningEffort,
			WorkingDirectory: provider.WorkingDirectory,
			TimeoutSeconds:   provider.TimeoutSeconds,
			Enabled:          &enabled,
		})
	}
	return normalized, rows, ids, nil
}

func validateExecutionCLIModelAndReasoning(provider CLIProviderInput) error {
	switch provider.Driver {
	case "codex":
		if _, exists := codexModels[provider.Model]; !exists {
			return executionConfigValidationErrorf("CLI provider「%s」的 Codex 模型不受支持", provider.ID)
		}
		if _, exists := codexReasoningEfforts[provider.ReasoningEffort]; !exists {
			return executionConfigValidationErrorf("CLI provider「%s」的 reasoning_effort 不受支持", provider.ID)
		}
	case "gemini":
		if _, exists := geminiModels[provider.Model]; !exists {
			return executionConfigValidationErrorf("CLI provider「%s」的 Gemini 模型不受支持", provider.ID)
		}
		if provider.ReasoningEffort != "" {
			return executionConfigValidationErrorf("CLI provider「%s」的 Gemini reasoning_effort 必须为空", provider.ID)
		}
	}
	return nil
}

func lowerASCII(value string) string {
	result := []byte(value)
	for index, character := range result {
		if character >= 'A' && character <= 'Z' {
			result[index] = character + ('a' - 'A')
		}
	}
	return string(result)
}

// NormalizeAIOrigin returns a stable origin key without relying on the Go
// toolchain's Unicode case tables. Hostname normalization is intentionally
// limited to ASCII A-Z; non-ASCII and invalid UTF-8 bytes remain unchanged.
func NormalizeAIOrigin(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Hostname() == "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", errors.New("invalid AI base URL")
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return lowerASCII(parsed.Scheme) + "://" + lowerASCII(parsed.Hostname()) + ":" + port, nil
}

func normalizedExecutionAIOrigin(raw string) (string, error) {
	return NormalizeAIOrigin(raw)
}

func upsertExecutionAPIProviders(tx *gorm.DB, rows []sysModel.AIProviderConfig) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"label", "type", "base_url", "api_key", "model", "max_tokens", "active", "updated_at",
		}),
	}).Create(&rows).Error
}

func upsertExecutionCLIProviders(tx *gorm.DB, rows []sysModel.CLIProviderConfig) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"label", "driver", "command_path", "model", "reasoning_effort",
			"working_directory", "timeout_seconds", "enabled", "updated_at",
		}),
	}).Create(&rows).Error
}

func deleteMissingExecutionAPIProviders(tx *gorm.DB, ids []string) error {
	if len(ids) == 0 {
		return tx.Where("1 = 1").Delete(&sysModel.AIProviderConfig{}).Error
	}
	return tx.Where("provider_id NOT IN ?", ids).Delete(&sysModel.AIProviderConfig{}).Error
}

func deleteMissingExecutionCLIProviders(tx *gorm.DB, ids []string) error {
	if len(ids) == 0 {
		return tx.Where("1 = 1").Delete(&sysModel.CLIProviderConfig{}).Error
	}
	return tx.Where("provider_id NOT IN ?", ids).Delete(&sysModel.CLIProviderConfig{}).Error
}

func providerIDsFromAPIRows(rows []sysModel.AIProviderConfig) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ProviderID)
	}
	return ids
}

func providerIDsFromCLIRows(rows []sysModel.CLIProviderConfig) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ProviderID)
	}
	return ids
}

func buildExecutionConfigResult(apiRows []sysModel.AIProviderConfig, cliRows []sysModel.CLIProviderConfig, target sysModel.SentenceExecutorConfig) ExecutionConfigInput {
	result := ExecutionConfigInput{
		ActiveTarget: ExecutionTarget{
			Type: strings.TrimSpace(target.ExecutorType),
			ID:   strings.TrimSpace(target.ExecutorID),
		},
		APIProviders: make([]AIProviderInput, 0, len(apiRows)),
		CLIProviders: make([]CLIProviderInput, 0, len(cliRows)),
	}
	for _, row := range apiRows {
		providerID := strings.TrimSpace(row.ProviderID)
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = providerID
		}
		result.APIProviders = append(result.APIProviders, AIProviderInput{
			ID:        providerID,
			Label:     label,
			Type:      strings.TrimSpace(row.Type),
			BaseURL:   strings.TrimRight(strings.TrimSpace(row.BaseURL), "/"),
			APIKey:    row.ApiKey,
			Model:     strings.TrimSpace(row.Model),
			MaxTokens: row.MaxTokens,
		})
	}
	for _, row := range cliRows {
		providerID := strings.TrimSpace(row.ProviderID)
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = providerID
		}
		enabled := true
		if row.Enabled != nil {
			enabled = *row.Enabled
		}
		result.CLIProviders = append(result.CLIProviders, CLIProviderInput{
			ID:               providerID,
			Label:            label,
			Driver:           strings.TrimSpace(row.Driver),
			CommandPath:      strings.TrimSpace(row.CommandPath),
			Model:            strings.TrimSpace(row.Model),
			ReasoningEffort:  strings.TrimSpace(row.ReasoningEffort),
			WorkingDirectory: strings.TrimSpace(row.WorkingDirectory),
			TimeoutSeconds:   row.TimeoutSeconds,
			Enabled:          enabled,
		})
	}
	return result
}

func validateLoadedExecutionTarget(input ExecutionConfigInput) error {
	switch input.ActiveTarget.Type {
	case "api":
		for _, provider := range input.APIProviders {
			if provider.ID == input.ActiveTarget.ID {
				return nil
			}
		}
		return fmt.Errorf("API 造句执行器「%s」不存在", input.ActiveTarget.ID)
	case "cli":
		for _, provider := range input.CLIProviders {
			if provider.ID != input.ActiveTarget.ID {
				continue
			}
			if !provider.Enabled {
				return fmt.Errorf("CLI 造句执行器「%s」已停用", input.ActiveTarget.ID)
			}
			return nil
		}
		return fmt.Errorf("CLI 造句执行器「%s」不存在", input.ActiveTarget.ID)
	default:
		return fmt.Errorf("造句执行器类型「%s」不存在", input.ActiveTarget.Type)
	}
}
