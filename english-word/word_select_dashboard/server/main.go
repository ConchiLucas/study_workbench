package main

import (
	"fmt"
	"os"

	"github.com/conchi/go-react-template/server/core"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/initialize"
	"github.com/conchi/go-react-template/server/utils"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

func main() {
	if err := run(initializeSystem, core.RunServer); err != nil {
		fmt.Fprintf(os.Stderr, "server startup failed: %v\n", err)
		os.Exit(1)
	}
}

func run(initialize func() error, serve func()) error {
	if err := initialize(); err != nil {
		return err
	}
	fmt.Printf("go-react-template v%s starting...\n", global.Version)
	serve()
	return nil
}

// initializeSystem 初始化系统所有组件
func initializeSystem() error {
	global.GVA_VP = core.Viper()
	initialize.OtherInit()
	global.GVA_LOG = core.Zap()
	zap.ReplaceGlobals(global.GVA_LOG)
	var err error
	global.GVA_DB, err = initialize.TryGorm()
	if err != nil {
		global.GVA_LOG.Warn("database unavailable during startup, continuing in initialization mode", zap.Error(err))
	}
	initialize.DBList()
	initialize.SetupHandlers()
	if global.GVA_DB != nil {
		if err := initialize.RegisterTables(); err != nil {
			return err
		}
		initialize.SyncAIConfigWithDatabase()
	}
	// 初始化 MinIO
	utils.InitMinio()
	return nil
}
