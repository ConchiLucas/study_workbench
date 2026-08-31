package initialize

import (
	"context"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/conchi/go-react-template/server/service/system"
	"gorm.io/gorm"
)

const initOrderEnsureTables = system.InitOrderExternal - 1

type ensureTables struct{}

func init() {
	system.RegisterInit(initOrderEnsureTables, &ensureTables{})
}

func (e *ensureTables) InitializerName() string {
	return "ensure_tables_created"
}

func (e *ensureTables) InitializeData(ctx context.Context) (next context.Context, err error) {
	return ctx, nil
}

func (e *ensureTables) DataInserted(ctx context.Context) bool {
	return true
}

func (e *ensureTables) MigrateTable(ctx context.Context) (context.Context, error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}
	tables := []interface{}{
		sysModel.SysUser{},
		sysModel.AIProviderConfig{},
		sysModel.TTSProviderConfig{},
		sysModel.CLIProviderConfig{},
		sysModel.SentenceExecutorConfig{},
	}
	for _, t := range tables {
		if err := db.AutoMigrate(&t); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (e *ensureTables) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	tables := []interface{}{
		sysModel.SysUser{},
		sysModel.AIProviderConfig{},
		sysModel.TTSProviderConfig{},
		sysModel.CLIProviderConfig{},
		sysModel.SentenceExecutorConfig{},
	}
	yes := true
	for _, t := range tables {
		yes = yes && db.Migrator().HasTable(t)
	}
	return yes
}
