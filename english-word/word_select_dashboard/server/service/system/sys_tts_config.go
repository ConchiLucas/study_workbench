package system

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	sysModel "github.com/conchi/go-react-template/server/model/system"
	"gorm.io/gorm"
)

const ttsProviderTypeMiMo = "mimo-tts"

const officialMiMoAPIHost = "api.xiaomimimo.com"

var (
	ErrInvalidTTSConfig     = errors.New("TTS 配置无效")
	ErrTTSConfigPersistence = errors.New("TTS 配置持久化失败")
)

type storedTTSProvider struct {
	apiKey  string
	baseURL string
}

type TTSConfigService struct{}

func (s *TTSConfigService) GetConfig(db *gorm.DB) (sysModel.TTSConfigPayload, error) {
	if db == nil {
		return sysModel.TTSConfigPayload{}, fmt.Errorf("%w: 数据库未连接", ErrTTSConfigPersistence)
	}

	var rows []sysModel.TTSProviderConfig
	if err := db.Order("provider_id ASC").Find(&rows).Error; err != nil {
		return sysModel.TTSConfigPayload{}, fmt.Errorf("%w: %v", ErrTTSConfigPersistence, err)
	}
	return buildSafeTTSConfig(rows), nil
}

func (s *TTSConfigService) SaveConfig(
	db *gorm.DB,
	input sysModel.TTSConfigPayload,
) (sysModel.TTSConfigPayload, error) {
	if db == nil {
		return sysModel.TTSConfigPayload{}, fmt.Errorf("%w: 数据库未连接", ErrTTSConfigPersistence)
	}

	var saved sysModel.TTSConfigPayload
	err := db.Transaction(func(tx *gorm.DB) error {
		var existingRows []sysModel.TTSProviderConfig
		if err := tx.Select("provider_id", "api_key", "base_url").Find(&existingRows).Error; err != nil {
			return fmt.Errorf("%w: %v", ErrTTSConfigPersistence, err)
		}
		existingProviders := make(map[string]storedTTSProvider, len(existingRows))
		for _, row := range existingRows {
			existingProviders[row.ProviderID] = storedTTSProvider{
				apiKey:  row.ApiKey,
				baseURL: row.BaseURL,
			}
		}

		rows, err := normalizeTTSConfig(input, existingProviders)
		if err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&sysModel.TTSProviderConfig{}).Error; err != nil {
			return fmt.Errorf("%w: %v", ErrTTSConfigPersistence, err)
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("%w: %v", ErrTTSConfigPersistence, err)
		}
		saved = buildSafeTTSConfig(rows)
		return nil
	})
	if err != nil {
		return sysModel.TTSConfigPayload{}, err
	}
	return saved, nil
}

func normalizeTTSConfig(
	input sysModel.TTSConfigPayload,
	existingProviders map[string]storedTTSProvider,
) ([]sysModel.TTSProviderConfig, error) {
	if len(input.Providers) == 0 {
		return nil, invalidTTSConfigf("请至少保留一个 TTS 配置")
	}

	activeID := strings.TrimSpace(input.Active)
	if activeID == "" {
		return nil, invalidTTSConfigf("请选择默认 TTS 配置")
	}

	seen := make(map[string]struct{}, len(input.Providers))
	rows := make([]sysModel.TTSProviderConfig, 0, len(input.Providers))
	activeFound := false
	for _, provider := range input.Providers {
		providerID := strings.TrimSpace(provider.ProviderID)
		if providerID == "" {
			return nil, invalidTTSConfigf("请填写 TTS 配置 ID")
		}
		if _, exists := seen[providerID]; exists {
			return nil, invalidTTSConfigf("TTS 配置 ID「%s」重复", providerID)
		}
		seen[providerID] = struct{}{}

		label := strings.TrimSpace(provider.Label)
		if label == "" {
			return nil, invalidTTSConfigf("请填写 TTS 配置「%s」的显示名称", providerID)
		}
		providerType := strings.TrimSpace(provider.Type)
		if providerType != ttsProviderTypeMiMo {
			return nil, invalidTTSConfigf("TTS 配置「%s」的类型不支持", providerID)
		}
		baseURL, err := normalizeTTSBaseURL(provider.BaseURL)
		if err != nil {
			return nil, invalidTTSConfigf("TTS 配置「%s」必须使用小米 MiMo 官方 HTTPS 地址", providerID)
		}
		model := strings.TrimSpace(provider.Model)
		if model == "" {
			return nil, invalidTTSConfigf("请填写 TTS 配置「%s」的模型名称", providerID)
		}
		voice := strings.TrimSpace(provider.Voice)
		if voice == "" {
			return nil, invalidTTSConfigf("请填写 TTS 配置「%s」的默认音色", providerID)
		}

		apiKey := strings.TrimSpace(provider.ApiKey)
		if apiKey == "" {
			stored := existingProviders[providerID]
			apiKey = strings.TrimSpace(stored.apiKey)
			if apiKey != "" && !sameTTSOrigin(baseURL, stored.baseURL) {
				return nil, invalidTTSConfigf(
					"TTS 配置「%s」的服务来源已变更，请重新填写 API Key",
					providerID,
				)
			}
		}
		if apiKey == "" {
			return nil, invalidTTSConfigf("请填写 TTS 配置「%s」的 API Key", providerID)
		}

		isActive := providerID == activeID
		if isActive {
			activeFound = true
			if !provider.Enabled {
				return nil, invalidTTSConfigf("默认 TTS 配置「%s」未启用", providerID)
			}
		}
		rows = append(rows, sysModel.TTSProviderConfig{
			ProviderID: providerID,
			Label:      label,
			Type:       providerType,
			BaseURL:    baseURL,
			ApiKey:     apiKey,
			Model:      model,
			Voice:      voice,
			Enabled:    provider.Enabled,
			Active:     isActive,
		})
	}

	if !activeFound {
		return nil, invalidTTSConfigf("默认 TTS 配置「%s」不存在", activeID)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ProviderID < rows[j].ProviderID
	})
	return rows, nil
}

func normalizeTTSBaseURL(raw string) (string, error) {
	normalized := strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(normalized)
	if err != nil ||
		parsed.Scheme != "https" ||
		!strings.EqualFold(parsed.Hostname(), officialMiMoAPIHost) ||
		(parsed.Port() != "" && parsed.Port() != "443") ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", fmt.Errorf("invalid TTS Base URL")
	}
	return normalized, nil
}

func sameTTSOrigin(left, right string) bool {
	leftURL, leftErr := url.Parse(strings.TrimSpace(left))
	rightURL, rightErr := url.Parse(strings.TrimSpace(right))
	if leftErr != nil || rightErr != nil {
		return false
	}
	return strings.EqualFold(leftURL.Scheme, rightURL.Scheme) &&
		strings.EqualFold(leftURL.Hostname(), rightURL.Hostname()) &&
		normalizedURLPort(leftURL) == normalizedURLPort(rightURL)
}

func normalizedURLPort(parsed *url.URL) string {
	if parsed.Port() != "" {
		return parsed.Port()
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(parsed.Scheme, "http") {
		return "80"
	}
	return ""
}

func invalidTTSConfigf(format string, args ...interface{}) error {
	return fmt.Errorf("%w：%s", ErrInvalidTTSConfig, fmt.Sprintf(format, args...))
}

func buildSafeTTSConfig(rows []sysModel.TTSProviderConfig) sysModel.TTSConfigPayload {
	response := sysModel.TTSConfigPayload{
		Providers: make([]sysModel.TTSProviderPayload, 0, len(rows)),
	}
	for _, row := range rows {
		if row.Active {
			response.Active = row.ProviderID
		}
		response.Providers = append(response.Providers, sysModel.TTSProviderPayload{
			ProviderID:       row.ProviderID,
			Label:            row.Label,
			Type:             row.Type,
			BaseURL:          row.BaseURL,
			ApiKey:           "",
			ApiKeyConfigured: strings.TrimSpace(row.ApiKey) != "",
			Model:            row.Model,
			Voice:            row.Voice,
			Enabled:          row.Enabled,
		})
	}
	return response
}
