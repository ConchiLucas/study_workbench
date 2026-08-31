package system

import "github.com/gin-gonic/gin"

type TTSConfigRouter struct{}

func (s *TTSConfigRouter) InitTTSConfigRouter(public, private *gin.RouterGroup) gin.IRoutes {
	public.Group("tts").GET("config", ttsConfigApi.GetConfig)
	privateRouter := private.Group("tts")
	privateRouter.POST("config", ttsConfigApi.SaveConfig)
	return privateRouter
}
