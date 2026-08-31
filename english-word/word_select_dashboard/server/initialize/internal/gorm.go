package internal

import (
	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/utils/gormsafe"
	"gorm.io/gorm"
)

var Gorm = new(_gorm)

type _gorm struct{}

// Config gorm 自定义配置
// Author [SliverHorn](https://github.com/SliverHorn)
func (g *_gorm) Config(general config.GeneralDB) *gorm.Config {
	return gormsafe.Config(general)
}
