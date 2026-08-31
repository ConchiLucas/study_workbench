package system

import "github.com/conchi/go-react-template/server/service"

type ApiGroup struct {
	DBApi
	BaseApi
	AIConfigApi
	ExecutionConfigApi
	TTSConfigApi
	SentenceApi
	ExecutionApi
	ClozeResultApi
	WordLibraryApi
	AppUserApi
}

var (
	userService            = service.ServiceGroupApp.SystemServiceGroup.UserService
	initDBService          = service.ServiceGroupApp.SystemServiceGroup.InitDBService
	aiConfigService        = service.ServiceGroupApp.SystemServiceGroup.AIConfigService
	executionConfigService = service.ServiceGroupApp.SystemServiceGroup.ExecutionConfigService
	ttsConfigService       = service.ServiceGroupApp.SystemServiceGroup.TTSConfigService
)
