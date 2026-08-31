package system

import "github.com/gin-gonic/gin"

type SentenceRouter struct{}

func (s *SentenceRouter) InitSentenceRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	sentenceRouter := Router.Group("sentences")
	{
		sentenceRouter.POST("generate", sentenceApi.Generate)
		sentenceRouter.GET("history", sentenceApi.History)
	}
	return sentenceRouter
}
