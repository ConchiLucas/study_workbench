package system

import "github.com/gin-gonic/gin"

type ExecutionRouter struct{}

func (s *ExecutionRouter) InitExecutionRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	executionRouter := Router.Group("executions")
	{
		executionRouter.GET("runs", executionApi.ListRuns)
	}
	return executionRouter
}
