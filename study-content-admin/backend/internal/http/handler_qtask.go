package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/conchi/study-content-admin/internal/qtask"
)

func (h *handlers) listQuestionTasks(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	subject := c.Query("subject")
	status := c.Query("status")
	res, err := h.deps.QTask.List(subject, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) createQuestionTask(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	var body struct {
		SubjectCode string `json:"subjectCode"`
		ModuleCode  string `json:"moduleCode"`
		Title       string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}
	task, err := h.deps.QTask.Create(qtask.CreateInput{
		SubjectCode: body.SubjectCode,
		ModuleCode:  body.ModuleCode,
		Title:       body.Title,
	})
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "模块不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *handlers) listQTaskLiteracyModules(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.QTask.ListLiteracyModules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) getQuestionTask(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return
	}
	task, err := h.deps.QTask.Get(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *handlers) reshuffleQuestionTask(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return
	}
	task, err := h.deps.QTask.Reshuffle(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *handlers) publishQuestionTask(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return
	}
	task, err := h.deps.QTask.Publish(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *handlers) unpublishQuestionTask(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return
	}
	task, err := h.deps.QTask.Unpublish(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *handlers) deleteQuestionTask(c *gin.Context) {
	if h.deps.QTask == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 id"})
		return
	}
	err = h.deps.QTask.Delete(id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err != nil {
		writeQTaskError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeQTaskError(c *gin.Context, err error) {
	msg := err.Error()
	status := http.StatusBadRequest
	if strings.Contains(msg, "可出题") || strings.Contains(msg, "draft") || strings.Contains(msg, "published") {
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{"error": msg})
}
