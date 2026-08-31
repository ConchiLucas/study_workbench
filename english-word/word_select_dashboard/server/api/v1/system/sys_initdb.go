package system

import (
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	"github.com/conchi/go-react-template/server/model/system"
	"github.com/conchi/go-react-template/server/model/system/request"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type DBApi struct{}

// InitDB
// @Tags     InitDB
// @Summary  初始化用户数据库
// @Produce  application/json
// @Param    data  body      request.InitDB                  true  "初始化数据库参数"
// @Success  200   {object}  response.Response{data=string}  "初始化用户数据库"
// @Router   /init/initdb [post]
func (i *DBApi) InitDB(c *gin.Context) {
	var dbInfo request.InitDB
	if err := c.ShouldBindJSON(&dbInfo); err != nil {
		global.GVA_LOG.Error("参数校验不通过!", zap.Error(err))
		response.FailWithMessage("参数校验不通过", c)
		return
	}
	if err := initDBService.InitDB(dbInfo); err != nil {
		global.GVA_LOG.Error("自动创建数据库失败!", zap.Error(err))
		response.FailWithMessage("自动创建数据库失败，请查看后台日志，检查后在进行初始化", c)
		return
	}
	response.OkWithMessage("自动创建数据库成功", c)
}

// CheckDB
// @Tags     CheckDB
// @Summary  初始化用户数据库
// @Produce  application/json
// @Success  200  {object}  response.Response{data=map[string]interface{},msg=string}  "初始化用户数据库"
// @Router   /init/checkdb [post]
func (i *DBApi) CheckDB(c *gin.Context) {
	var (
		message     = "前往初始化数据库"
		needInit    = true
		needMigrate = false
	)

	if global.GVA_DB != nil {
		message = "数据库无需初始化"
		needInit = false
		// 检查是否有缺失的表结构需要迁移
		missingTables := []string{}
		tableChecks := map[string]interface{}{
			"sys_users": system.SysUser{},
			// 在此添加更多表结构检查
		}
		for name, model := range tableChecks {
			if !global.GVA_DB.Migrator().HasTable(model) {
				missingTables = append(missingTables, name)
			}
		}
		if len(missingTables) > 0 {
			needMigrate = true
			message = "数据库已初始化，但存在未同步的表结构"
		}
	}
	global.GVA_LOG.Info(message)
	response.OkWithDetailed(gin.H{"needInit": needInit, "needMigrate": needMigrate}, message, c)
}

// MigrateTables
// @Tags     InitDB
// @Summary  同步新增表结构
// @Produce  application/json
// @Success  200  {object}  response.Response{msg=string}  "同步表结构"
// @Router   /init/migrateTables [post]
func (i *DBApi) MigrateTables(c *gin.Context) {
	if global.GVA_DB == nil {
		response.FailWithMessage("数据库未连接，请先初始化数据库", c)
		return
	}
	err := global.GVA_DB.AutoMigrate(
		system.SysUser{},
		// 在此添加更多表结构
	)
	if err != nil {
		global.GVA_LOG.Error("同步表结构失败!", zap.Error(err))
		response.FailWithMessage("同步表结构失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("同步表结构成功", c)
}
