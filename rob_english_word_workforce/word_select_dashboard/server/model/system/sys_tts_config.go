package system

import "time"

type TTSProviderConfig struct {
	ID         uint      `json:"ID" gorm:"primarykey"`
	ProviderID string    `json:"providerId" gorm:"uniqueIndex;size:120;not null;comment:TTS 配置 ID"`
	Label      string    `json:"label" gorm:"size:160;not null;comment:TTS 配置名称"`
	Type       string    `json:"type" gorm:"size:80;not null;comment:TTS 接口类型"`
	BaseURL    string    `json:"baseUrl" gorm:"column:base_url;size:500;not null;comment:Base URL"`
	ApiKey     string    `json:"-" gorm:"column:api_key;type:text;not null;comment:API Key"`
	Model      string    `json:"model" gorm:"size:160;not null;comment:TTS 模型名称"`
	Voice      string    `json:"voice" gorm:"size:120;not null;comment:默认音色"`
	Enabled    bool      `json:"enabled" gorm:"not null;comment:是否启用"`
	Active     bool      `json:"active" gorm:"index;not null;comment:是否默认配置"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (TTSProviderConfig) TableName() string {
	return "tts_provider_configs"
}

type TTSProviderPayload struct {
	ProviderID       string `json:"id"`
	Label            string `json:"label"`
	Type             string `json:"type"`
	BaseURL          string `json:"base_url"`
	ApiKey           string `json:"api_key"`
	ApiKeyConfigured bool   `json:"api_key_configured"`
	Model            string `json:"model"`
	Voice            string `json:"voice"`
	Enabled          bool   `json:"enabled"`
}

type TTSConfigPayload struct {
	Active    string               `json:"active"`
	Providers []TTSProviderPayload `json:"providers"`
}
