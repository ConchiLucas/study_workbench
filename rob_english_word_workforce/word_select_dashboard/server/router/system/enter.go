package system

import api "github.com/conchi/go-react-template/server/api/v1"

type RouterGroup struct {
	BaseRouter
	InitRouter
	UserRouter
	AIConfigRouter
	ExecutionConfigRouter
	TTSConfigRouter
	SentenceRouter
	ExecutionRouter
	ClozeResultRouter
	WordLibraryRouter
	AppUserRouter
}

var (
	dbApi              = api.ApiGroupApp.SystemApiGroup.DBApi
	baseApi            = api.ApiGroupApp.SystemApiGroup.BaseApi
	aiConfigApi        = api.ApiGroupApp.SystemApiGroup.AIConfigApi
	executionConfigApi = api.ApiGroupApp.SystemApiGroup.ExecutionConfigApi
	ttsConfigApi       = api.ApiGroupApp.SystemApiGroup.TTSConfigApi
	sentenceApi        = api.ApiGroupApp.SystemApiGroup.SentenceApi
	executionApi       = api.ApiGroupApp.SystemApiGroup.ExecutionApi
	clozeResultApi     = api.ApiGroupApp.SystemApiGroup.ClozeResultApi
	wordLibraryApi     = api.ApiGroupApp.SystemApiGroup.WordLibraryApi
	appUserApi         = api.ApiGroupApp.SystemApiGroup.AppUserApi
)
