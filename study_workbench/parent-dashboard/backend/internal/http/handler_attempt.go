package http

import (
	"github.com/gin-gonic/gin"

	"github.com/conchi/study-workbench/internal/service"
)

type reportAttemptsReq struct {
	Attempts []service.AttemptInput `json:"attempts" binding:"required,min=1,dive"`
}

func (h *handlers) postAttempts(c *gin.Context) {
	if h.deps.Attempt == nil {
		fail(c, 500, "attempt service not ready")
		return
	}
	childID, ok := pathInt64(c, "cid")
	if !ok {
		return
	}
	var req reportAttemptsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err.Error())
		return
	}
	states, err := h.deps.Attempt.Report(childID, req.Attempts)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, gin.H{"states": states})
}

func (h *handlers) markMastered(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	kpID, ok2 := pathInt64(c, "kpId")
	if !ok1 || !ok2 {
		return
	}
	st, err := h.deps.Attempt.MarkMastered(childID, kpID)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, st)
}

func (h *handlers) undoMark(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	kpID, ok2 := pathInt64(c, "kpId")
	if !ok1 || !ok2 {
		return
	}
	st, err := h.deps.Attempt.UndoMark(childID, kpID)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, st)
}
