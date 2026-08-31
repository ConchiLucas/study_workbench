package http

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/conchi/study-workbench/internal/service"
)

func (h *handlers) getTodayPlan(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Plan.Today(childID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

// createTodayPlan 家长端显式生成今天的任务；内部与 GET today 相同（幂等）。
func (h *handlers) createTodayPlan(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Plan.Today(childID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) createExtraPlan(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Plan.Extra(childID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) startPlan(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	planID, ok2 := pathInt64(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	data, err := h.deps.Plan.Start(childID, planID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) answerPlanItem(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	planID, ok2 := pathInt64(c, "planId")
	itemID, ok3 := pathInt64(c, "itemId")
	if !ok1 || !ok2 || !ok3 {
		return
	}
	var in service.AnswerInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, err.Error())
		return
	}
	data, err := h.deps.Plan.Answer(childID, planID, itemID, in)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) finishPlan(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	planID, ok2 := pathInt64(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	data, err := h.deps.Plan.Finish(childID, planID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) listPlans(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Plan.History(childID, c.Query("from"), c.Query("to"), c.Query("status"))
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) listTodoPlans(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Plan.Todo(childID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) getPlan(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	planID, ok2 := pathInt64(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	data, err := h.deps.Plan.Detail(childID, planID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

func (h *handlers) reviewPlan(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	planID, ok2 := pathInt64(c, "planId")
	if !ok1 || !ok2 {
		return
	}
	data, err := h.deps.Plan.Review(childID, planID)
	if err != nil {
		failPlan(c, err)
		return
	}
	ok200(c, data)
}

// failPlan 把业务错误映射到合适的状态码，别让"计划不存在"也返回 500。
func failPlan(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPlanNotFound), errors.Is(err, service.ErrPlanItemNotFound):
		fail(c, 404, err.Error())
	case errors.Is(err, service.ErrTooManyPlansToday):
		fail(c, 409, err.Error())
	case errors.Is(err, service.ErrNoCandidates):
		fail(c, 422, err.Error())
	default:
		fail(c, 500, err.Error())
	}
}
