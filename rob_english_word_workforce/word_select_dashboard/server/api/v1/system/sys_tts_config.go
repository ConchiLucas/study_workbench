package system

import (
	"errors"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	sysService "github.com/conchi/go-react-template/server/service/system"
	"github.com/gin-gonic/gin"
)

type TTSConfigApi struct{}

func (a *TTSConfigApi) GetConfig(c *gin.Context) {
	config, err := ttsConfigService.GetConfig(global.GVA_DB)
	if err != nil {
		response.FailWithMessage("读取 TTS 配置失败", c)
		return
	}
	response.OkWithDetailed(config, "获取成功", c)
}

func (a *TTSConfigApi) SaveConfig(c *gin.Context) {
	var input sysModel.TTSConfigPayload
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FailWithMessage("TTS 配置格式错误", c)
		return
	}

	config, err := ttsConfigService.SaveConfig(global.GVA_DB, input)
	if err != nil {
		if errors.Is(err, sysService.ErrInvalidTTSConfig) {
			response.FailWithMessage(err.Error(), c)
			return
		}
		response.FailWithMessage("保存 TTS 配置失败", c)
		return
	}
	response.OkWithDetailed(config, "保存成功", c)
}
