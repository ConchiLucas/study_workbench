package gormsafe

import (
	"time"

	"github.com/conchi/go-react-template/server/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func ConfigWithWriter(general config.GeneralDB, writer logger.Writer) *gorm.Config {
	return &gorm.Config{
		Logger: logger.New(writer, logger.Config{
			SlowThreshold:        200 * time.Millisecond,
			LogLevel:             general.LogLevel(),
			Colorful:             true,
			ParameterizedQueries: true,
		}),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   general.Prefix,
			SingularTable: general.Singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}
}
