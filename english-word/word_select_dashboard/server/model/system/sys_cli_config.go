package system

import "time"

type CLIProviderConfig struct {
	ID               uint      `json:"ID" gorm:"primarykey"`
	ProviderID       string    `json:"providerId" gorm:"uniqueIndex;size:120;not null"`
	Label            string    `json:"label" gorm:"size:160;not null"`
	Driver           string    `json:"driver" gorm:"size:32;not null"`
	CommandPath      string    `json:"commandPath" gorm:"type:text;not null"`
	Model            string    `json:"model" gorm:"size:160;not null"`
	ReasoningEffort  string    `json:"reasoningEffort" gorm:"size:32;not null;default:''"`
	WorkingDirectory string    `json:"workingDirectory" gorm:"type:text;not null"`
	TimeoutSeconds   int       `json:"timeoutSeconds" gorm:"not null;default:300"`
	Enabled          *bool     `json:"enabled" gorm:"not null;default:true"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (CLIProviderConfig) TableName() string {
	return "cli_provider_configs"
}

type SentenceExecutorConfig struct {
	SingletonKey string    `json:"singletonKey" gorm:"primarykey;size:32"`
	ExecutorType string    `json:"executorType" gorm:"size:16;not null"`
	ExecutorID   string    `json:"executorId" gorm:"size:120;not null"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (SentenceExecutorConfig) TableName() string {
	return "sentence_executor_config"
}
