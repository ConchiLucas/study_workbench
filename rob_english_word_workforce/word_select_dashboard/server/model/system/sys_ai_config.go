package system

import "time"

type AIProviderConfig struct {
	ID         uint      `json:"ID" gorm:"primarykey"`
	ProviderID string    `json:"providerId" gorm:"uniqueIndex;size:120;not null;comment:AI 配置 ID"`
	Label      string    `json:"label" gorm:"size:160;comment:模型配置名称"`
	Type       string    `json:"type" gorm:"size:80;not null;comment:模型兼容类型"`
	BaseURL    string    `json:"baseUrl" gorm:"column:base_url;size:500;not null;comment:Base URL"`
	ApiKey     string    `json:"-" gorm:"column:api_key;type:text;comment:API Key"`
	Model      string    `json:"model" gorm:"size:160;not null;comment:模型名称"`
	MaxTokens  int       `json:"maxTokens" gorm:"comment:最大 token 数"`
	Active     bool      `json:"active" gorm:"index;comment:是否默认配置"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (AIProviderConfig) TableName() string {
	return "ai_provider_configs"
}
