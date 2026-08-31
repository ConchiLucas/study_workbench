package system

import "github.com/gin-gonic/gin"

type AIConfigRouter struct{}

func (s *AIConfigRouter) InitAIConfigRouter(public, private *gin.RouterGroup) gin.IRoutes {
	public.Group("ai").GET("config", aiConfigApi.GetConfig)
	privateRouter := private.Group("ai")
	privateRouter.POST("config", aiConfigApi.SaveConfig)
	return privateRouter
}
