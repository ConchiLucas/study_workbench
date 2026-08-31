package system

import (
	"fmt"
	"sort"
	"strings"

	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	sysService "github.com/conchi/go-react-template/server/service/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AIConfigApi struct{}

var aiConfigGetSnapshotHook func()

type AIProviderConfigItem struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Type             string `json:"type"`
	BaseURL          string `json:"base_url"`
	ApiKey           string `json:"api_key"`
	ApiKeyConfigured bool   `json:"api_key_configured"`
	Model            string `json:"model"`
	MaxTokens        int    `json:"max_tokens"`
}

type AIConfigRequest struct {
	Active    string                 `json:"active"`
	Providers []AIProviderConfigItem `json:"providers"`
}

type storedAIProvider struct {
	apiKey  string
	baseURL string
}

func (a *AIConfigApi) GetConfig(c *gin.Context) {
	aiConfig, err := a.loadConfigSnapshot(true)
	if err != nil {
		global.GVA_LOG.Warn("读取数据库 AI 配置失败", zap.Error(err))
		response.FailWithMessage("读取 AI 配置失败", c)
		return
	}
	if aiConfigGetSnapshotHook != nil {
		aiConfigGetSnapshotHook()
	}

	providers := make([]AIProviderConfigItem, 0, len(aiConfig.Providers))
	for _, provider := range aiConfig.ListProviders() {
		providers = append(providers, AIProviderConfigItem{
			ID:               provider.ID,
			Label:            provider.Label,
			Type:             provider.Type,
			BaseURL:          provider.BaseURL,
			ApiKey:           "",
			ApiKeyConfigured: strings.TrimSpace(provider.ApiKey) != "",
			Model:            provider.Model,
			MaxTokens:        provider.MaxTokens,
		})
	}

	response.OkWithDetailed(AIConfigRequest{
		Active:    aiConfig.Active,
		Providers: providers,
	}, "获取成功", c)
}

func (a *AIConfigApi) loadConfigSnapshot(publishDatabaseConfig bool) (config.AI, error) {
	var publish func(config.AI)
	if publishDatabaseConfig {
		publish = func(aiConfig config.AI) {
			global.GVA_CONFIG.AI = aiConfig
			if global.GVA_VP != nil {
				global.GVA_VP.Set("ai.active", aiConfig.Active)
				global.GVA_VP.Set("ai.providers", aiConfig.Providers)
			}
		}
	}
	snapshot, _, err := aiConfigService.ReadConfigSnapshot(global.GVA_DB, func() config.AI {
		return global.GVA_CONFIG.AI
	}, publish)
	return snapshot, err
}

func (a *AIConfigApi) SaveConfig(c *gin.Context) {
	var req AIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	existingProviders, err := a.loadExistingAIProviders()
	if err != nil {
		global.GVA_LOG.Error("读取已有 AI API Key 失败", zap.Error(err))
		response.FailWithMessage("读取已有 AI 配置失败", c)
		return
	}
	aiConfig, normalizedReq, err := normalizeAIConfig(req, existingProviders)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	effectiveAIConfig, err := aiConfigService.SaveConfigAndPublish(global.GVA_DB, aiConfig, func(committed config.AI) {
		global.GVA_VP.Set("ai.active", committed.Active)
		global.GVA_VP.Set("ai.providers", committed.Providers)
		global.GVA_CONFIG.AI = committed
		if err := global.GVA_VP.WriteConfig(); err != nil {
			// Database is the source of truth. Keep the committed runtime state and
			// let the next startup repopulate Viper memory from the database.
			global.GVA_LOG.Warn("AI 配置已保存到数据库，但写入配置文件失败", zap.Error(err))
		}
	})
	if err != nil {
		global.GVA_LOG.Error("保存 AI 配置到数据库失败", zap.Error(err))
		response.FailWithMessage("保存数据库失败: "+err.Error(), c)
		return
	}

	normalizedReq.Active = effectiveAIConfig.Active
	response.OkWithDetailed(normalizedReq, "保存成功", c)
}

func (a *AIConfigApi) loadExistingAIProviders() (map[string]storedAIProvider, error) {
	snapshot, err := a.loadConfigSnapshot(false)
	if err != nil {
		return nil, err
	}
	providers := make(map[string]storedAIProvider)
	for id, provider := range snapshot.Providers {
		if apiKey := strings.TrimSpace(provider.ApiKey); apiKey != "" {
			providers[id] = storedAIProvider{apiKey: apiKey, baseURL: provider.BaseURL}
		}
	}
	if len(snapshot.Providers) == 0 {
		if apiKey := strings.TrimSpace(snapshot.ApiKey); apiKey != "" {
			providers["default"] = storedAIProvider{apiKey: apiKey, baseURL: snapshot.BaseURL}
		}
	}
	return providers, nil
}

func normalizeAIConfig(req AIConfigRequest, existingProviders map[string]storedAIProvider) (config.AI, AIConfigRequest, error) {
	providers := make(map[string]config.AIProvider, len(req.Providers))
	normalizedProviders := make([]AIProviderConfigItem, 0, len(req.Providers))
	active := strings.TrimSpace(req.Active)

	for _, item := range req.Providers {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return config.AI{}, AIConfigRequest{}, fmt.Errorf("请填写 AI 配置 ID")
		}
		if _, exists := providers[id]; exists {
			return config.AI{}, AIConfigRequest{}, fmt.Errorf("AI 配置 ID「%s」重复", id)
		}

		providerType := strings.TrimSpace(item.Type)
		if providerType == "" {
			providerType = config.AIProviderTypeOpenAICompatible
		}
		if providerType != config.AIProviderTypeOpenAICompatible && providerType != config.AIProviderTypeAnthropicCompatible {
			return config.AI{}, AIConfigRequest{}, fmt.Errorf("AI 配置「%s」的类型不支持", id)
		}

		baseURL := strings.TrimRight(strings.TrimSpace(item.BaseURL), "/")
		if baseURL == "" {
			return config.AI{}, AIConfigRequest{}, fmt.Errorf("请填写 AI 配置「%s」的 Base URL", id)
		}
		origin, err := sysService.NormalizeAIOrigin(baseURL)
		if err != nil {
			return config.AI{}, AIConfigRequest{}, fmt.Errorf("AI 配置「%s」的 Base URL 无效", id)
		}
		model := strings.TrimSpace(item.Model)
		if model == "" {
			return config.AI{}, AIConfigRequest{}, fmt.Errorf("请填写 AI 配置「%s」的模型名称", id)
		}
		maxTokens := item.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 4096
		}

		apiKey := strings.TrimSpace(item.ApiKey)
		if apiKey == "" {
			stored, exists := existingProviders[id]
			if !exists || strings.TrimSpace(stored.apiKey) == "" {
				return config.AI{}, AIConfigRequest{}, fmt.Errorf("请填写 AI 配置「%s」的 API Key", id)
			}
			storedOrigin, storedOriginErr := sysService.NormalizeAIOrigin(stored.baseURL)
			if storedOriginErr != nil || origin != storedOrigin {
				return config.AI{}, AIConfigRequest{}, fmt.Errorf(
					"AI 配置「%s」的服务来源已变更，请重新填写 API Key",
					id,
				)
			}
			apiKey = strings.TrimSpace(stored.apiKey)
		}
		provider := config.AIProvider{
			Label:     strings.TrimSpace(item.Label),
			Type:      providerType,
			BaseURL:   baseURL,
			ApiKey:    apiKey,
			Model:     model,
			MaxTokens: maxTokens,
		}
		providers[id] = provider
		normalizedProviders = append(normalizedProviders, AIProviderConfigItem{
			ID:               id,
			Label:            provider.Label,
			Type:             provider.Type,
			BaseURL:          provider.BaseURL,
			ApiKey:           "",
			ApiKeyConfigured: provider.ApiKey != "",
			Model:            provider.Model,
			MaxTokens:        provider.MaxTokens,
		})
	}

	if len(providers) == 0 {
		return config.AI{}, AIConfigRequest{}, fmt.Errorf("请至少保留一个 AI 配置")
	}
	if active == "" {
		active = firstAIConfigProviderID(providers)
	}
	if _, exists := providers[active]; !exists {
		return config.AI{}, AIConfigRequest{}, fmt.Errorf("默认 AI 配置「%s」不存在", active)
	}

	sort.SliceStable(normalizedProviders, func(i, j int) bool {
		return normalizedProviders[i].ID < normalizedProviders[j].ID
	})

	return config.AI{Active: active, Providers: providers}, AIConfigRequest{
		Active:    active,
		Providers: normalizedProviders,
	}, nil
}

func firstAIConfigProviderID(providers map[string]config.AIProvider) string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}
