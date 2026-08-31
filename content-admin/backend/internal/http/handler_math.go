package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *handlers) syncMath(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	res, err := h.deps.Math.Sync()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) listMath(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	view := c.DefaultQuery("view", "groups")
	res, err := h.deps.Math.List(view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) generateMathGlyph(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Math.GenerateGlyph(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *handlers) serveMathGlyphPNG(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	png, err := h.deps.Math.GlyphPNG(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", png)
}

func (h *handlers) batchMathGlyphs(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	moduleCode := strings.TrimSpace(c.Query("moduleCode"))
	if moduleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moduleCode 不能为空"})
		return
	}
	res, err := h.deps.Math.BatchGenerateGlyphs(c.Request.Context(), moduleCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handlers) serveMathSpeechMP3(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	mp3, err := h.deps.Math.SpeechMP3(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
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

func (h *handlers) regenerateMathSpeech(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	kpID, err := strconv.ParseInt(c.Param("kpId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 kpId"})
		return
	}
	dto, err := h.deps.Math.RegenerateSpeech(c.Request.Context(), kpID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在，请先同步"})
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

func (h *handlers) batchMathSpeech(c *gin.Context) {
	if h.deps.Math == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "数据库未配置"})
		return
	}
	moduleCode := strings.TrimSpace(c.Query("moduleCode"))
	if moduleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moduleCode 不能为空"})
		return
	}
	res, err := h.deps.Math.BatchGenerateSpeech(c.Request.Context(), moduleCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
