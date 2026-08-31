package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) syncEnglish(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.English.Sync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) listEnglish(c *gin.Context) {
	if h.deps.English == nil {
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
	res, err := h.deps.English.List(view, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) patchEnglishWord(c *gin.Context) {
	if h.deps.English == nil {
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
	dto, err := h.deps.English.PatchOverride(kpID, body.NeedsSenseImageOverride)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) generateEnglishGlyph(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.English.GenerateGlyph(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchEnglishGlyphs(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	force := c.Query("force") == "1" || c.Query("force") == "true"
	res, err := h.deps.English.BatchGenerateGlyphs(c.Request.Context(), force)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveEnglishGlyphPNG(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.English.GlyphPNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *handlers) serveEnglishSpeechMP3(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	mp3, err := h.deps.English.SpeechMP3(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在，请先同步"})
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

func (h *handlers) regenerateEnglishSpeech(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.English.RegenerateSpeech(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在，请先同步"})
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

func (h *handlers) batchEnglishSpeech(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	moduleCode := strings.TrimSpace(c.Query("moduleCode"))
	if moduleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moduleCode 不能为空"})
		return
	}
	res, err := h.deps.English.BatchGenerateSpeech(c.Request.Context(), moduleCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) generateEnglishSense(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.English.GenerateSense(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) batchEnglishSenses(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	workers, _ := strconv.Atoi(c.DefaultQuery("workers", "0"))
	maxRetries, _ := strconv.Atoi(c.DefaultQuery("maxRetries", "3"))
	res, err := h.deps.English.BatchGenerateSenses(c.Request.Context(), c.Query("moduleCode"), workers, maxRetries)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveEnglishSensePNG(c *gin.Context) {
	if h.deps.English == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.English.SensePNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "词不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}
