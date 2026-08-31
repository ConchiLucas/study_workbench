package system

import "github.com/gin-gonic/gin"

type AppUserRouter struct{}

func (s *AppUserRouter) InitAppUserRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	appUserRouter := Router.Group("users")
	{
		appUserRouter.GET("", appUserApi.List)
		appUserRouter.GET("wrong-words", appUserApi.WrongWords)
		appUserRouter.GET("cloze-wrong-words", appUserApi.ClozeWrongWords)
		appUserRouter.GET("mastered-words", appUserApi.MasteredWords)
		appUserRouter.GET(":userId/training-results", appUserApi.TrainingResults)
	}
	return appUserRouter
}
