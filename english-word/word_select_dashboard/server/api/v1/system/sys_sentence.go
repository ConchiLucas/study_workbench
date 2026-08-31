package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	systemModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SentenceApi struct{}

type GenerateSentenceRequest struct {
	Words []string `json:"words"`
}

type GenerateSentenceResponse struct {
	RunID         string   `json:"runId"`
	Status        string   `json:"status"`
	Words         []string `json:"words"`
	Sentence      string   `json:"sentence"`
	TranslationZh string   `json:"translationZh"`
	ExplanationZh string   `json:"explanationZh"`
	ProviderID    string   `json:"providerId"`
	ProviderLabel string   `json:"providerLabel"`
	Model         string   `json:"model"`
	DurationMs    int64    `json:"durationMs"`
}

type SentenceHistoryResponse struct {
	RunID         string    `json:"runId"`
	Status        string    `json:"status"`
	Words         []string  `json:"words"`
	Sentence      string    `json:"sentence"`
	TranslationZh string    `json:"translationZh"`
	ExplanationZh string    `json:"explanationZh"`
	ProviderID    string    `json:"providerId"`
	ProviderLabel string    `json:"providerLabel"`
	Model         string    `json:"model"`
	DurationMs    int64     `json:"durationMs"`
	CreatedAt     time.Time `json:"createdAt"`
}

type wordAgentSentenceResponse struct {
	Sentence      string   `json:"sentence"`
	TranslationZh string   `json:"translationZh"`
	ExplanationZh string   `json:"explanationZh"`
	Words         []string `json:"words"`
	ProviderID    string   `json:"providerId"`
	ProviderLabel string   `json:"providerLabel"`
	Model         string   `json:"model"`
}

func (a *SentenceApi) Generate(c *gin.Context) {
	startedAt := time.Now()
	runID := fmt.Sprintf("run-%s-%s", startedAt.Format("20060102150405"), uuid.NewString()[:8])

	var req GenerateSentenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	words, err := normalizeSentenceWords(req.Words)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	agentResp, err := callWordAgentGenerateSentence(words)
	durationMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		saveErr := saveSentenceExecutionRun(startedAt, runID, durationMs, err)
		if saveErr != nil {
			err = fmt.Errorf("%v；保存执行记录失败: %w", err, saveErr)
		}
		response.FailWithDetailed(GenerateSentenceResponse{
			RunID:      runID,
			Status:     "failed",
			Words:      words,
			DurationMs: durationMs,
		}, err.Error(), c)
		return
	}

	responseWords := agentResp.Words
	if len(responseWords) == 0 {
		responseWords = words
	}

	if err := saveSentenceSuccessResult(startedAt, runID, words, agentResp, durationMs); err != nil {
		response.FailWithDetailed(GenerateSentenceResponse{
			RunID:         runID,
			Status:        "success",
			Words:         responseWords,
			Sentence:      agentResp.Sentence,
			TranslationZh: agentResp.TranslationZh,
			ExplanationZh: agentResp.ExplanationZh,
			ProviderID:    agentResp.ProviderID,
			ProviderLabel: agentResp.ProviderLabel,
			Model:         agentResp.Model,
			DurationMs:    durationMs,
		}, "生成成功，但保存记录失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(GenerateSentenceResponse{
		RunID:         runID,
		Status:        "success",
		Words:         responseWords,
		Sentence:      agentResp.Sentence,
		TranslationZh: agentResp.TranslationZh,
		ExplanationZh: agentResp.ExplanationZh,
		ProviderID:    agentResp.ProviderID,
		ProviderLabel: agentResp.ProviderLabel,
		Model:         agentResp.Model,
		DurationMs:    durationMs,
	}, "生成成功", c)
}

func (a *SentenceApi) History(c *gin.Context) {
	if global.GVA_DB == nil {
		response.FailWithMessage("数据库未初始化", c)
		return
	}

	pageStr := c.Query("page")
	pageSizeStr := c.Query("pageSize")

	if pageStr != "" {
		page, _ := strconv.Atoi(pageStr)
		if page <= 0 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(pageSizeStr)
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageSize > 500 {
			pageSize = 500
		}

		var total int64
		if err := global.GVA_DB.Model(&systemModel.SentenceGenerationRecord{}).Count(&total).Error; err != nil {
			response.FailWithMessage("获取造句记录总数失败: "+err.Error(), c)
			return
		}

		var records []systemModel.SentenceGenerationRecord
		if err := global.GVA_DB.Order("created_at desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&records).Error; err != nil {
			response.FailWithMessage("获取造句记录失败: "+err.Error(), c)
			return
		}

		runIDs := make([]string, 0, len(records))
		for _, record := range records {
			runIDs = append(runIDs, record.RunID)
		}

		executionByRunID := make(map[string]systemModel.ExecutionRun, len(runIDs))
		if len(runIDs) > 0 {
			var executions []systemModel.ExecutionRun
			if err := global.GVA_DB.Where("run_id IN ?", runIDs).Find(&executions).Error; err != nil {
				response.FailWithMessage("获取执行记录失败: "+err.Error(), c)
				return
			}
			for _, execution := range executions {
				executionByRunID[execution.RunID] = execution
			}
		}

		items := make([]SentenceHistoryResponse, 0, len(records))
		for _, record := range records {
			execution, exists := executionByRunID[record.RunID]
			status := systemModel.ExecutionStatusSuccess
			var durationMs int64
			if exists {
				status = execution.Status
				durationMs = execution.DurationMs
			}

			items = append(items, SentenceHistoryResponse{
				RunID:         record.RunID,
				Status:        status,
				Words:         decodeSentenceWords(record.Words),
				Sentence:      record.Sentence,
				TranslationZh: record.TranslationZh,
				ExplanationZh: record.ExplanationZh,
				ProviderID:    record.ProviderID,
				ProviderLabel: record.ProviderLabel,
				Model:         record.Model,
				DurationMs:    durationMs,
				CreatedAt:     record.CreatedAt,
			})
		}

		response.OkWithDetailed(response.PageResult{
			List:     items,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, "获取成功", c)
		return
	}

	limit := parseExecutionLimit(c.Query("limit"))
	var records []systemModel.SentenceGenerationRecord
	if err := global.GVA_DB.Order("created_at desc").Limit(limit).Find(&records).Error; err != nil {
		response.FailWithMessage("获取造句记录失败: "+err.Error(), c)
		return
	}

	runIDs := make([]string, 0, len(records))
	for _, record := range records {
		runIDs = append(runIDs, record.RunID)
	}

	executionByRunID := make(map[string]systemModel.ExecutionRun, len(runIDs))
	if len(runIDs) > 0 {
		var executions []systemModel.ExecutionRun
		if err := global.GVA_DB.Where("run_id IN ?", runIDs).Find(&executions).Error; err != nil {
			response.FailWithMessage("获取执行记录失败: "+err.Error(), c)
			return
		}
		for _, execution := range executions {
			executionByRunID[execution.RunID] = execution
		}
	}

	items := make([]SentenceHistoryResponse, 0, len(records))
	for _, record := range records {
		execution, exists := executionByRunID[record.RunID]
		status := systemModel.ExecutionStatusSuccess
		var durationMs int64
		if exists {
			status = execution.Status
			durationMs = execution.DurationMs
		}

		items = append(items, SentenceHistoryResponse{
			RunID:         record.RunID,
			Status:        status,
			Words:         decodeSentenceWords(record.Words),
			Sentence:      record.Sentence,
			TranslationZh: record.TranslationZh,
			ExplanationZh: record.ExplanationZh,
			ProviderID:    record.ProviderID,
			ProviderLabel: record.ProviderLabel,
			Model:         record.Model,
			DurationMs:    durationMs,
			CreatedAt:     record.CreatedAt,
		})
	}

	response.OkWithDetailed(items, "获取成功", c)
}

func normalizeSentenceWords(words []string) ([]string, error) {
	cleanedWords := make([]string, 0, len(words))
	seen := make(map[string]struct{}, len(words))
	for _, word := range words {
		cleanedWord := strings.TrimSpace(word)
		if cleanedWord == "" {
			continue
		}
		key := strings.ToLower(cleanedWord)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleanedWords = append(cleanedWords, cleanedWord)
	}

	if len(cleanedWords) == 0 {
		return nil, fmt.Errorf("请至少输入一个单词")
	}
	if len(cleanedWords) > 12 {
		return nil, fmt.Errorf("一次最多支持 12 个单词")
	}

	return cleanedWords, nil
}

func callWordAgentGenerateSentence(words []string) (wordAgentSentenceResponse, error) {
	baseURL := global.GVA_CONFIG.WordAgent.ResolveBaseURL()
	requestBody, err := json.Marshal(GenerateSentenceRequest{Words: words})
	if err != nil {
		return wordAgentSentenceResponse{}, fmt.Errorf("请求参数编码失败: %w", err)
	}

	client := http.Client{Timeout: global.GVA_CONFIG.WordAgent.Timeout()}
	req, err := http.NewRequest(
		http.MethodPost,
		baseURL+"/v1/sentences/generate",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return wordAgentSentenceResponse{}, fmt.Errorf("创建 Python 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return wordAgentSentenceResponse{}, fmt.Errorf("调用 Python word-agent 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return wordAgentSentenceResponse{}, fmt.Errorf("读取 Python 响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return wordAgentSentenceResponse{}, fmt.Errorf("Python word-agent 返回失败: %s", extractWordAgentError(body))
	}

	var agentResp wordAgentSentenceResponse
	if err := json.Unmarshal(body, &agentResp); err != nil {
		return wordAgentSentenceResponse{}, fmt.Errorf("解析 Python 响应失败: %w", err)
	}
	if strings.TrimSpace(agentResp.Sentence) == "" {
		return wordAgentSentenceResponse{}, fmt.Errorf("Python word-agent 没有返回句子")
	}

	return agentResp, nil
}

func saveSentenceExecutionRun(
	startedAt time.Time,
	runID string,
	durationMs int64,
	runErr error,
) error {
	if global.GVA_DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	return createSentenceExecutionRun(global.GVA_DB, startedAt, runID, durationMs, runErr)
}

func saveSentenceSuccessResult(
	startedAt time.Time,
	runID string,
	words []string,
	agentResp wordAgentSentenceResponse,
	durationMs int64,
) error {
	if global.GVA_DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		if err := createSentenceExecutionRun(tx, startedAt, runID, durationMs, nil); err != nil {
			return err
		}
		return createSentenceGenerationRecord(tx, runID, words, agentResp)
	})
}

func createSentenceExecutionRun(
	db *gorm.DB,
	startedAt time.Time,
	runID string,
	durationMs int64,
	runErr error,
) error {
	finishedAt := time.Now()
	status := systemModel.ExecutionStatusSuccess
	currentStepID := "save_result"
	errorMessage := ""
	if runErr != nil {
		status = systemModel.ExecutionStatusFailed
		currentStepID = "execute_business"
		errorMessage = runErr.Error()
	}

	record := systemModel.ExecutionRun{
		RunID:         runID,
		BusinessType:  systemModel.ExecutionBusinessSentenceGeneration,
		Title:         "单词造句",
		Status:        status,
		CurrentStepID: currentStepID,
		DurationMs:    durationMs,
		Error:         errorMessage,
		StartedAt:     startedAt.UnixMilli(),
		FinishedAt:    finishedAt.UnixMilli(),
	}

	return db.Create(&record).Error
}

func createSentenceGenerationRecord(db *gorm.DB, runID string, words []string, agentResp wordAgentSentenceResponse) error {
	recordWords := words
	if len(agentResp.Words) > 0 {
		recordWords = agentResp.Words
	}
	wordsJSON, err := json.Marshal(recordWords)
	if err != nil {
		return fmt.Errorf("造句单词编码失败: %w", err)
	}

	record := systemModel.SentenceGenerationRecord{
		RunID:         runID,
		Words:         string(wordsJSON),
		Sentence:      agentResp.Sentence,
		TranslationZh: agentResp.TranslationZh,
		ExplanationZh: agentResp.ExplanationZh,
		ProviderID:    agentResp.ProviderID,
		ProviderLabel: agentResp.ProviderLabel,
		Model:         agentResp.Model,
	}

	return db.Create(&record).Error
}

func decodeSentenceWords(raw string) []string {
	var words []string
	if err := json.Unmarshal([]byte(raw), &words); err != nil {
		return []string{}
	}
	return words
}

func extractWordAgentError(body []byte) string {
	var payload struct {
		Detail any `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Detail != nil {
		return fmt.Sprint(payload.Detail)
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "空响应"
	}
	return text
}
