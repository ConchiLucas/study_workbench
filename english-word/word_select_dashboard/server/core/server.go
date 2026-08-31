package core

import (
	"fmt"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/initialize"
	"go.uber.org/zap"
	"time"
)

func RunServer() {
	if global.GVA_CONFIG.System.UseRedis {
		// 初始化redis服务
		initialize.Redis()
		if global.GVA_CONFIG.System.UseMultipoint {
			initialize.RedisList()
		}
	}

	if global.GVA_CONFIG.System.UseMongo {
		err := initialize.Mongo.Initialization()
		if err != nil {
			zap.L().Error(fmt.Sprintf("%+v", err))
		}
	}

	Router := initialize.Routers()

	address := fmt.Sprintf(":%d", global.GVA_CONFIG.System.Addr)

	fmt.Printf(`
	欢迎使用 go-react-template
	当前版本:%s
	默认前端文件运行地址:http://127.0.0.1%s
`, global.Version, address)
	initServer(address, Router, 10*time.Minute, 10*time.Minute)
}
