package system

import (
	"strconv"
	"strings"
	"time"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	systemModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/gin-gonic/gin"
)

type ExecutionApi struct{}

type ExecutionRunResponse struct {
	RunID         string    `json:"runId"`
	BusinessType  string    `json:"businessType"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`
	CurrentStepID string    `json:"currentStepId"`
	DurationMs    int64     `json:"durationMs"`
	Error         string    `json:"error"`
	StartedAt     time.Time `json:"startedAt"`
	FinishedAt    time.Time `json:"finishedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (a *ExecutionApi) ListRuns(c *gin.Context) {
	if global.GVA_DB == nil {
		response.FailWithMessage("数据库未初始化", c)
		return
	}

	page, pageSize := parseExecutionPagination(c)
	businessType := strings.TrimSpace(c.Query("businessType"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	startedAtFrom := parseExecutionUnixMilli(c.Query("startedAtFrom"))
	startedAtTo := parseExecutionUnixMilli(c.Query("startedAtTo"))

	db := global.GVA_DB.Model(&systemModel.ExecutionRun{})
	if businessType != "" {
		db = db.Where("business_type = ?", businessType)
	}
	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		db = db.Where(
			"run_id ILIKE ? OR business_type ILIKE ? OR title ILIKE ? OR current_step_id ILIKE ? OR status ILIKE ? OR error ILIKE ?",
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
		)
	}
	if startedAtFrom > 0 {
		db = db.Where("started_at >= ?", startedAtFrom)
	}
	if startedAtTo > 0 {
		db = db.Where("started_at <= ?", startedAtTo)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		response.FailWithMessage("获取执行记录总数失败: "+err.Error(), c)
		return
	}

	var records []systemModel.ExecutionRun
	if err := db.Order("started_at desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&records).Error; err != nil {
		response.FailWithMessage("获取执行记录失败: "+err.Error(), c)
		return
	}

	items := make([]ExecutionRunResponse, 0, len(records))
	for _, record := range records {
		items = append(items, mapExecutionRun(record))
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func parseExecutionPagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(strings.TrimSpace(c.Query("page")))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(strings.TrimSpace(c.Query("pageSize")))
	if pageSize <= 0 {
		pageSize = parseExecutionLimit(c.Query("limit"))
	}
	if pageSize > 500 {
		pageSize = 500
	}
	return page, pageSize
}

func parseExecutionLimit(raw string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit <= 0 {
		return 10
	}
	return limit
}

func parseExecutionUnixMilli(raw string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func mapExecutionRun(record systemModel.ExecutionRun) ExecutionRunResponse {
	return ExecutionRunResponse{
		RunID:         record.RunID,
		BusinessType:  record.BusinessType,
		Title:         record.Title,
		Status:        record.Status,
		CurrentStepID: record.CurrentStepID,
		DurationMs:    record.DurationMs,
		Error:         record.Error,
		StartedAt:     unixMilliToTime(record.StartedAt),
		FinishedAt:    unixMilliToTime(record.FinishedAt),
		CreatedAt:     record.CreatedAt,
	}
}

func unixMilliToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value)
}
