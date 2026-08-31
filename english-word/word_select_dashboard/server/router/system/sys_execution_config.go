package system

import "github.com/gin-gonic/gin"

type ExecutionConfigRouter struct{}

func (s *ExecutionConfigRouter) InitExecutionConfigRouter(public, private *gin.RouterGroup) gin.IRoutes {
	_ = public
	privateRouter := private.Group("ai")
	privateRouter.GET("execution-config", executionConfigApi.GetConfig)
	privateRouter.POST("execution-config", executionConfigApi.SaveConfig)
	return privateRouter
}
