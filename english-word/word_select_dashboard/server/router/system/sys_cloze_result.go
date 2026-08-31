package system

import "github.com/gin-gonic/gin"

type ClozeResultRouter struct{}

func (s *ClozeResultRouter) InitClozeResultRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	clozeResultRouter := Router.Group("cloze-results")
	{
		clozeResultRouter.GET("users", clozeResultApi.Users)
		clozeResultRouter.GET("items", clozeResultApi.Items)
	}
	return clozeResultRouter
}
