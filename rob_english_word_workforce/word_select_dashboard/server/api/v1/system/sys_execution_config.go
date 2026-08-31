package system

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"strings"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	sysService "github.com/conchi/go-react-template/server/service/system"
	"github.com/gin-gonic/gin"
)

type ExecutionConfigApi struct{}

type ExecutionAIProviderResponse struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Type             string `json:"type"`
	BaseURL          string `json:"base_url"`
	APIKey           string `json:"api_key"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	Model            string `json:"model"`
	MaxTokens        int    `json:"max_tokens"`
}

type ExecutionConfigResponse struct {
	ActiveTarget sysService.ExecutionTarget     `json:"active_target"`
	APIProviders []ExecutionAIProviderResponse  `json:"api_providers"`
	CLIProviders []ExecutionCLIProviderResponse `json:"cli_providers"`
}

type ExecutionCLIProviderResponse struct {
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

func (a *ExecutionConfigApi) GetConfig(c *gin.Context) {
	config, err := executionConfigService.Load(global.GVA_DB)
	if err != nil {
		response.FailWithMessage("读取造句执行器配置失败", c)
		return
	}
	response.OkWithDetailed(buildExecutionConfigResponse(config), "获取成功", c)
}

func (a *ExecutionConfigApi) SaveConfig(c *gin.Context) {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.FailWithMessage("参数错误: Content-Type 必须为 application/json", c)
		return
	}

	var input sysService.ExecutionConfigInput
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("请求体只能包含一个 JSON 值")
		}
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	config, err := executionConfigService.SaveAndLoad(global.GVA_DB, input)
	if err != nil {
		if sysService.IsExecutionConfigValidationError(err) {
			response.FailWithMessage(err.Error(), c)
			return
		}
		response.FailWithMessage("保存造句执行器配置失败", c)
		return
	}
	response.OkWithDetailed(buildExecutionConfigResponse(config), "保存成功", c)
}

func buildExecutionConfigResponse(input sysService.ExecutionConfigInput) ExecutionConfigResponse {
	result := ExecutionConfigResponse{
		ActiveTarget: input.ActiveTarget,
		APIProviders: make([]ExecutionAIProviderResponse, 0, len(input.APIProviders)),
		CLIProviders: make([]ExecutionCLIProviderResponse, 0, len(input.CLIProviders)),
	}
	for _, provider := range input.APIProviders {
		result.APIProviders = append(result.APIProviders, ExecutionAIProviderResponse{
			ID:               provider.ID,
			Label:            provider.Label,
			Type:             provider.Type,
			BaseURL:          provider.BaseURL,
			APIKey:           "",
			APIKeyConfigured: strings.TrimSpace(provider.APIKey) != "",
			Model:            provider.Model,
			MaxTokens:        provider.MaxTokens,
		})
	}
	for _, provider := range input.CLIProviders {
		result.CLIProviders = append(result.CLIProviders, ExecutionCLIProviderResponse{
			ID:               provider.ID,
			Label:            provider.Label,
			Driver:           provider.Driver,
			CommandPath:      provider.CommandPath,
			Model:            provider.Model,
			ReasoningEffort:  provider.ReasoningEffort,
			WorkingDirectory: provider.WorkingDirectory,
			TimeoutSeconds:   provider.TimeoutSeconds,
			Enabled:          provider.Enabled,
		})
	}
	return result
}
