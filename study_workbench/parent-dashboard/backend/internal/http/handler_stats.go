package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *handlers) getTrend(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	data, err := h.deps.Stats.Trend(childID, days)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) getCalendar(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	months, _ := strconv.Atoi(c.DefaultQuery("months", "3"))
	data, err := h.deps.Stats.Calendar(childID, months)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) getReviewQueue(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	data, err := h.deps.Stats.ReviewQueue(childID, limit)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}
