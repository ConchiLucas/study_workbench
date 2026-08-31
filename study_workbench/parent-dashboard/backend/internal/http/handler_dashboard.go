package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *handlers) getOverview(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Dashboard.Overview(childID)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) getSubjects(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Dashboard.Subjects(childID)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) getMatrix(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Dashboard.Matrix(childID, c.Query("subject"))
	if err != nil {
		fail(c, 400, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) getAttention(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	data, err := h.deps.Dashboard.Attention(childID, limit)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) getKpDetail(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	kpID, ok2 := pathInt64(c, "kpId")
	if !ok1 || !ok2 {
		return
	}
	data, err := h.deps.Dashboard.KpDetail(childID, kpID)
	if err != nil {
		fail(c, 404, err.Error())
		return
	}
	ok200(c, data)
}
