package system

import "github.com/gin-gonic/gin"

type WordLibraryRouter struct{}

func (s *WordLibraryRouter) InitWordLibraryRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	wordLibraryRouter := Router.Group("word-libraries")
	{
		wordLibraryRouter.GET("", wordLibraryApi.Libraries)
		wordLibraryRouter.GET("clean-words", wordLibraryApi.CleanWords)
		wordLibraryRouter.GET("clean-words/:id/sentences", wordLibraryApi.CleanWordSentences)
		wordLibraryRouter.POST("clean-sentences/score", wordLibraryApi.ScoreCleanSentences)
		wordLibraryRouter.GET(":id/words", wordLibraryApi.Words)
	}
	return wordLibraryRouter
}
