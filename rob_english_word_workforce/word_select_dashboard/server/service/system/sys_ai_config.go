package system

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/conchi/go-react-template/server/config"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AIConfigService struct{}

func (s *AIConfigService) SaveConfig(db *gorm.DB, aiConfig config.AI) error {
	if db == nil {
		return errors.New("数据库未连接，无法保存 AI 配置")
	}
	return withExecutionConfigWriteCoordinator(func() error {
		_, err := saveAIConfig(db, aiConfig)
		return err
	})
}

func (s *AIConfigService) SaveConfigAndPublish(
	db *gorm.DB,
	aiConfig config.AI,
	publish func(config.AI),
) (config.AI, error) {
	if db == nil {
		return config.AI{}, errors.New("数据库未连接，无法保存 AI 配置")
	}

	var effective config.AI
	err := withExecutionConfigWriteCoordinator(func() error {
		var err error
		effective, err = saveAIConfig(db, aiConfig)
		if err != nil {
			return err
		}
		if publish != nil {
			// Publication callbacks run while the coordinator is held. They must be
			// fast and must not re-enter AI/execution config service methods.
			publish(cloneAIConfig(effective))
		}
		return nil
	})
	if err != nil {
		return config.AI{}, err
	}
	return cloneAIConfig(effective), nil
}

func (s *AIConfigService) RefreshConfigFromDatabaseAndPublish(
	db *gorm.DB,
	publish func(config.AI),
) (bool, error) {
	_, refreshed, err := s.ReadConfigSnapshot(db, nil, publish)
	return refreshed, err
}

func (s *AIConfigService) ReadConfigSnapshot(
	db *gorm.DB,
	fallback func() config.AI,
	publishDatabaseConfig func(config.AI),
) (config.AI, bool, error) {
	var snapshot config.AI
	var fromDatabase bool
	err := withExecutionConfigWriteCoordinator(func() error {
		if db != nil {
			hasDatabaseConfig, err := s.HasDatabaseConfig(db)
			if err != nil {
				return err
			}
			if hasDatabaseConfig {
				loaded, found, err := s.LoadConfig(db)
				if err != nil {
					return err
				}
				if !found {
					loaded = config.AI{Providers: map[string]config.AIProvider{}}
				}
				snapshot = cloneAIConfig(loaded)
				if publishDatabaseConfig != nil {
					// Publication callbacks run while the coordinator is held. They must
					// be fast and must not re-enter config service methods.
					publishDatabaseConfig(cloneAIConfig(loaded))
				}
				fromDatabase = true
				return nil
			}
		}

		if fallback != nil {
			snapshot = cloneAIConfig(fallback())
		} else {
			snapshot = config.AI{Providers: map[string]config.AIProvider{}}
		}
		return nil
	})
	if err != nil {
		return config.AI{}, false, err
	}
	return snapshot, fromDatabase, nil
}

func cloneAIConfig(source config.AI) config.AI {
	providers := make(map[string]config.AIProvider, len(source.Providers))
	for id, provider := range source.Providers {
		providers[id] = provider
	}
	return config.AI{
		Active:    source.Active,
		Providers: providers,
		BaseURL:   source.BaseURL,
		ApiKey:    source.ApiKey,
		Model:     source.Model,
		MaxTokens: source.MaxTokens,
	}
}

func saveAIConfig(db *gorm.DB, aiConfig config.AI) (config.AI, error) {
	rows, err := buildAIProviderConfigRows(aiConfig)
	if err != nil {
		return config.AI{}, err
	}
	providerIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		providerIDs = append(providerIDs, row.ProviderID)
	}

	var effective config.AI
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := acquireExecutionConfigGuard(tx); err != nil {
			return err
		}
		target, targetExists, err := findExecutionTarget(tx, "UPDATE")
		if err != nil {
			return err
		}
		if targetExists {
			switch strings.TrimSpace(target.ExecutorType) {
			case "cli":
				for index := range rows {
					rows[index].Active = false
				}
			case "api":
				targetID := strings.TrimSpace(target.ExecutorID)
				foundTarget := false
				for index := range rows {
					rows[index].Active = rows[index].ProviderID == targetID
					foundTarget = foundTarget || rows[index].Active
				}
				if !foundTarget {
					return fmt.Errorf("当前造句执行器 API provider「%s」不能删除", targetID)
				}
			default:
				return fmt.Errorf("造句执行器类型「%s」不存在", target.ExecutorType)
			}
		}
		if err := tx.Where("provider_id NOT IN ?", providerIDs).Delete(&sysModel.AIProviderConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "provider_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"label",
				"type",
				"base_url",
				"api_key",
				"model",
				"max_tokens",
				"active",
				"updated_at",
			}),
		}).Create(&rows).Error; err != nil {
			return err
		}
		effective = buildAIConfigFromRows(rows)
		return nil
	})
	if err != nil {
		return config.AI{}, err
	}
	return effective, nil
}

func buildAIConfigFromRows(rows []sysModel.AIProviderConfig) config.AI {
	providers := make(map[string]config.AIProvider, len(rows))
	active := ""
	for _, row := range rows {
		providerID := strings.TrimSpace(row.ProviderID)
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = providerID
		}
		providers[providerID] = config.AIProvider{
			Label:     label,
			Type:      strings.TrimSpace(row.Type),
			BaseURL:   strings.TrimRight(strings.TrimSpace(row.BaseURL), "/"),
			ApiKey:    strings.TrimSpace(row.ApiKey),
			Model:     strings.TrimSpace(row.Model),
			MaxTokens: row.MaxTokens,
		}
		if row.Active {
			active = providerID
		}
	}
	return config.AI{Active: active, Providers: providers}
}

func (s *AIConfigService) HasDatabaseConfig(db *gorm.DB) (bool, error) {
	if db == nil {
		return false, errors.New("数据库未连接，无法检查 AI 配置")
	}

	var providerCount int64
	if err := db.Model(&sysModel.AIProviderConfig{}).Limit(1).Count(&providerCount).Error; err != nil {
		return false, err
	}
	if providerCount > 0 {
		return true, nil
	}
	_, found, err := findExecutionTarget(db, "")
	return found, err
}

func (s *AIConfigService) LoadConfig(db *gorm.DB) (config.AI, bool, error) {
	if db == nil {
		return config.AI{}, false, errors.New("数据库未连接，无法读取 AI 配置")
	}

	var loaded config.AI
	var found bool
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := acquireExecutionConfigGuard(tx); err != nil {
			return err
		}
		target, targetExists, err := findExecutionTarget(tx, "SHARE")
		if err != nil {
			return err
		}
		var rows []sysModel.AIProviderConfig
		if err := tx.Order("provider_id ASC").Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		providers := make(map[string]config.AIProvider, len(rows))
		legacyActive := ""
		for _, row := range rows {
			providerID := strings.TrimSpace(row.ProviderID)
			if providerID == "" {
				continue
			}
			label := strings.TrimSpace(row.Label)
			if label == "" {
				label = providerID
			}
			providers[providerID] = config.AIProvider{
				Label:     label,
				Type:      strings.TrimSpace(row.Type),
				BaseURL:   strings.TrimRight(strings.TrimSpace(row.BaseURL), "/"),
				ApiKey:    strings.TrimSpace(row.ApiKey),
				Model:     strings.TrimSpace(row.Model),
				MaxTokens: row.MaxTokens,
			}
			if row.Active && legacyActive == "" {
				legacyActive = providerID
			}
		}
		if len(providers) == 0 {
			return nil
		}

		active := legacyActive
		if targetExists {
			switch strings.TrimSpace(target.ExecutorType) {
			case "cli":
				active = ""
			case "api":
				active = strings.TrimSpace(target.ExecutorID)
				if _, exists := providers[active]; !exists {
					return fmt.Errorf("当前造句执行器 API provider「%s」不存在", active)
				}
			default:
				return fmt.Errorf("造句执行器类型「%s」不存在", target.ExecutorType)
			}
		} else if active == "" {
			active = firstAIProviderID(providers)
		}
		loaded = config.AI{Active: active, Providers: providers}
		found = true
		return nil
	})
	if err != nil {
		return config.AI{}, false, err
	}
	return loaded, found, nil
}

func (s *AIConfigService) HasUsableConfig(aiConfig config.AI) bool {
	for _, provider := range aiConfig.ListProviders() {
		if strings.TrimSpace(provider.BaseURL) != "" && strings.TrimSpace(provider.Model) != "" {
			return true
		}
	}
	return false
}

func buildAIProviderConfigRows(aiConfig config.AI) ([]sysModel.AIProviderConfig, error) {
	providers := aiConfig.ListProviders()
	if len(providers) == 0 {
		return nil, errors.New("AI 配置为空，无法保存到数据库")
	}

	rows := make([]sysModel.AIProviderConfig, 0, len(providers))
	for _, provider := range providers {
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" {
			return nil, errors.New("AI 配置 ID 不能为空")
		}
		maxTokens := provider.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 4096
		}
		rows = append(rows, sysModel.AIProviderConfig{
			ProviderID: providerID,
			Label:      strings.TrimSpace(provider.Label),
			Type:       strings.TrimSpace(provider.Type),
			BaseURL:    strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/"),
			ApiKey:     strings.TrimSpace(provider.ApiKey),
			Model:      strings.TrimSpace(provider.Model),
			MaxTokens:  maxTokens,
			Active:     provider.Active,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].ProviderID < rows[j].ProviderID
	})
	return rows, nil
}

func firstAIProviderID(providers map[string]config.AIProvider) string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}
