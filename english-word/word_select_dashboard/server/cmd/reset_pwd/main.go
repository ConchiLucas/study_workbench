package main

import (
	"fmt"
	"github.com/conchi/go-react-template/server/core"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/initialize"
	"github.com/conchi/go-react-template/server/utils"
	"go.uber.org/zap"
)

func main() {
	global.GVA_VP = core.Viper()
	initialize.OtherInit()
	global.GVA_LOG = core.Zap()
	zap.ReplaceGlobals(global.GVA_LOG)
	global.GVA_DB = initialize.Gorm()

	newHash := utils.BcryptHash("123456")
	result := global.GVA_DB.Exec("UPDATE sys_users SET password = ? WHERE username = 'admin'", newHash)
	fmt.Printf("密码已重置为 123456, 影响行数: %d\n", result.RowsAffected)
}
