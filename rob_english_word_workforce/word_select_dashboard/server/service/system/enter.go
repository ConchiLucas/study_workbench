package system

type ServiceGroup struct {
	UserService
	InitDBService
	AIConfigService
	ExecutionConfigService
	TTSConfigService
}
