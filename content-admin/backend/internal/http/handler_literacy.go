package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) syncLiteracy(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.Literacy.Sync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) listLiteracy(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	view := c.DefaultQuery("view", "groups")
	var filter *bool
	switch c.Query("needsSenseImage") {
	case "true", "1":
		v := true
		filter = &v
	case "false", "0":
		v := false
		filter = &v
	}
	res, err := h.deps.Literacy.List(view, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) patchLiteracyChar(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	var body struct {
		NeedsSenseImageOverride *bool `json:"needsSenseImageOverride"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}
	// Allow explicit null to clear override: client sends {"needsSenseImageOverride": null}
	dto, err := h.deps.Literacy.PatchOverride(kpID, body.NeedsSenseImageOverride)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) generateGlyph(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Literacy.GenerateGlyph(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchGlyphs(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.Literacy.BatchGenerateGlyphs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveGlyphPNG(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.Literacy.GlyphPNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *handlers) serveSpeechMP3(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	mp3, err := h.deps.Literacy.SpeechMP3(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在，请先同步"})
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusBadRequest
		if strings.Contains(msg, "TTS 未就绪") || strings.Contains(msg, "加载语音配置失败") || strings.Contains(msg, "没有可用的 TTS") || strings.Contains(msg, "语音存储未就绪") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "audio/mpeg", mp3)
}

func (h *handlers) regenerateSpeech(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Literacy.RegenerateSpeech(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在，请先同步"})
		return
	}
	if err != nil {
		msg := err.Error()
		status := http.StatusBadRequest
		if strings.Contains(msg, "TTS 未就绪") || strings.Contains(msg, "加载语音配置失败") || strings.Contains(msg, "没有可用的 TTS") || strings.Contains(msg, "语音存储未就绪") {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchSpeech(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	moduleCode := strings.TrimSpace(c.Query("moduleCode"))
	if moduleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moduleCode 不能为空"})
		return
	}
	res, err := h.deps.Literacy.BatchGenerateSpeech(c.Request.Context(), moduleCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) generateSense(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Literacy.GenerateSense(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchSenses(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	workers, _ := strconv.Atoi(c.DefaultQuery("workers", "0"))
	maxRetries, _ := strconv.Atoi(c.DefaultQuery("maxRetries", "3"))
	res, err := h.deps.Literacy.BatchGenerateSenses(c.Request.Context(), c.Query("moduleCode"), workers, maxRetries)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveSensePNG(c *gin.Context) {
	if h.deps.Literacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.Literacy.SensePNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "字不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}
