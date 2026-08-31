package http

import (
	"github.com/gin-gonic/gin"

	"github.com/conchi/study-workbench/internal/service"
)

type Deps struct {
	Attempt   *service.AttemptService
	Dashboard *service.DashboardService
	Stats     *service.StatsService
	Reward    *service.RewardService
	Plan      *service.PlanService
}

type handlers struct{ deps Deps }

func NewRouter(d Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), cors(), currentUser())
	h := &handlers{deps: d}

	r.GET("/healthz", func(c *gin.Context) { ok200(c, gin.H{"status": "ok"}) })

	v1 := r.Group("/api/v1")
	{
		child := v1.Group("/children/:cid")
		child.POST("/attempts", h.postAttempts)
		child.POST("/knowledge-points/:kpId/mark", h.markMastered)
		child.DELETE("/knowledge-points/:kpId/mark", h.undoMark)

		child.GET("/overview", h.getOverview)
		child.GET("/subjects", h.getSubjects)
		child.GET("/mastery/matrix", h.getMatrix)
		child.GET("/attention", h.getAttention)
		child.GET("/knowledge-points/:kpId", h.getKpDetail)

		child.GET("/stats/trend", h.getTrend)
		child.GET("/stats/calendar", h.getCalendar)
		child.GET("/review-queue", h.getReviewQueue)

		child.GET("/rewards", h.listRewards)
		child.POST("/rewards/:rewardId/redeem", h.redeemReward)

		child.GET("/plans/today", h.getTodayPlan)
		child.POST("/plans/today", h.createTodayPlan)
		child.GET("/plans/todo", h.listTodoPlans)
		child.GET("/plans", h.listPlans)
		child.POST("/plans/extra", h.createExtraPlan)
		child.GET("/plans/:planId", h.getPlan)
		child.GET("/plans/:planId/review", h.reviewPlan)
		child.POST("/plans/:planId/start", h.startPlan)
		child.POST("/plans/:planId/items/:itemId/answer", h.answerPlanItem)
		child.POST("/plans/:planId/finish", h.finishPlan)
	}
	mountSPA(r)
	return r
}
