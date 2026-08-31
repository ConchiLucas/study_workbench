package initialize

import (
	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/service"
	"go.uber.org/zap"
)

func SyncAIConfigWithDatabase() {
	if global.GVA_DB == nil {
		return
	}

	aiConfigService := service.ServiceGroupApp.SystemServiceGroup.AIConfigService
	hasDatabaseConfig, err := aiConfigService.HasDatabaseConfig(global.GVA_DB)
	if err != nil {
		clearRuntimeAIConfig()
		global.GVA_LOG.Warn("failed to inspect database AI config", zap.Error(err))
		return
	}
	if hasDatabaseConfig {
		loadRuntimeAIConfigFromDatabase()
		return
	}

	if !aiConfigService.HasUsableConfig(global.GVA_CONFIG.AI) {
		return
	}
	if err := aiConfigService.SaveConfig(global.GVA_DB, global.GVA_CONFIG.AI); err != nil {
		global.GVA_LOG.Warn("failed to seed database AI config", zap.Error(err))
		return
	}
	loadRuntimeAIConfigFromDatabase()
	global.GVA_LOG.Info("seeded database AI config")
}

func loadRuntimeAIConfigFromDatabase() {
	aiConfigService := service.ServiceGroupApp.SystemServiceGroup.AIConfigService
	aiConfig, found, err := aiConfigService.LoadConfig(global.GVA_DB)
	if err != nil {
		clearRuntimeAIConfig()
		global.GVA_LOG.Warn("failed to load AI config from database", zap.Error(err))
		return
	}
	if !found {
		aiConfig = config.AI{Providers: map[string]config.AIProvider{}}
	}

	global.GVA_CONFIG.AI = aiConfig
	if global.GVA_VP != nil {
		global.GVA_VP.Set("ai.active", aiConfig.Active)
		global.GVA_VP.Set("ai.providers", aiConfig.Providers)
	}
	global.GVA_LOG.Info("loaded AI config from database")
}

func clearRuntimeAIConfig() {
	empty := config.AI{Providers: map[string]config.AIProvider{}}
	global.GVA_CONFIG.AI = empty
	if global.GVA_VP != nil {
		global.GVA_VP.Set("ai.active", empty.Active)
		global.GVA_VP.Set("ai.providers", empty.Providers)
	}
}
