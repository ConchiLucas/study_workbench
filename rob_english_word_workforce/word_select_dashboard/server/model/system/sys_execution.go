package system

import "github.com/conchi/go-react-template/server/global"

const (
	ExecutionBusinessSentenceGeneration = "sentence_generation"

	ExecutionStatusSuccess = "success"
	ExecutionStatusFailed  = "failed"
)

type ExecutionRun struct {
	global.GVA_MODEL
	RunID         string `json:"runId" gorm:"uniqueIndex;size:80;not null;comment:执行ID"`
	BusinessType  string `json:"businessType" gorm:"index;size:80;not null;comment:业务类型"`
	Title         string `json:"title" gorm:"size:120;comment:执行标题"`
	Status        string `json:"status" gorm:"index;size:32;not null;comment:执行状态"`
	CurrentStepID string `json:"currentStepId" gorm:"size:80;comment:当前步骤ID"`
	DurationMs    int64  `json:"durationMs" gorm:"comment:耗时毫秒"`
	Error         string `json:"error" gorm:"type:text;comment:错误信息"`
	StartedAt     int64  `json:"startedAt" gorm:"index;comment:开始时间Unix毫秒"`
	FinishedAt    int64  `json:"finishedAt" gorm:"comment:完成时间Unix毫秒"`
}

func (ExecutionRun) TableName() string {
	return "execution_runs"
}

type SentenceGenerationRecord struct {
	global.GVA_MODEL
	RunID         string `json:"runId" gorm:"uniqueIndex;size:80;not null;comment:执行ID"`
	Words         string `json:"words" gorm:"type:text;comment:输入单词JSON"`
	Sentence      string `json:"sentence" gorm:"type:text;comment:生成句子"`
	TranslationZh string `json:"translationZh" gorm:"type:text;comment:中文翻译"`
	ExplanationZh string `json:"explanationZh" gorm:"type:text;comment:中文解释"`
	ProviderID    string `json:"providerId" gorm:"size:120;comment:模型配置ID"`
	ProviderLabel string `json:"providerLabel" gorm:"size:160;comment:模型配置名称"`
	Model         string `json:"model" gorm:"size:160;comment:模型名称"`
}

func (SentenceGenerationRecord) TableName() string {
	return "sentence_generation_records"
}
