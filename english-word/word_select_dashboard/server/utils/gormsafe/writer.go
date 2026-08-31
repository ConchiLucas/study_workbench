package gormsafe

import (
	"fmt"
	"sync"

	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Writer struct {
	config config.GeneralDB
}

func NewWriter(general config.GeneralDB) *Writer {
	return &Writer{config: general}
}

func (w *Writer) Printf(message string, data ...interface{}) {
	fmt.Printf(message, data...)
	if !w.config.LogZap {
		return
	}
	formatted := fmt.Sprintf(message, data...)
	switch w.config.LogLevel() {
	case logger.Silent:
		global.GVA_LOG.Debug(formatted)
	case logger.Error:
		global.GVA_LOG.Error(formatted)
	case logger.Warn:
		global.GVA_LOG.Warn(formatted)
	case logger.Info:
		global.GVA_LOG.Info(formatted)
	default:
		global.GVA_LOG.Info(formatted)
	}
}

type WriterFactory func(config.GeneralDB) logger.Writer

type writerFactoryOverride struct {
	id      uint64
	factory WriterFactory
}

var writerFactoryState struct {
	sync.RWMutex
	nextID    uint64
	overrides []writerFactoryOverride
}

func defaultWriterFactory(general config.GeneralDB) logger.Writer {
	return NewWriter(general)
}

func Config(general config.GeneralDB) *gorm.Config {
	writerFactoryState.RLock()
	factory := defaultWriterFactory
	if count := len(writerFactoryState.overrides); count > 0 {
		factory = writerFactoryState.overrides[count-1].factory
	}
	writerFactoryState.RUnlock()
	return ConfigWithWriter(general, factory(general))
}

func SetWriterFactoryForTesting(factory WriterFactory) func() {
	if factory == nil {
		panic("gormsafe: nil writer factory")
	}
	writerFactoryState.Lock()
	writerFactoryState.nextID++
	id := writerFactoryState.nextID
	writerFactoryState.overrides = append(writerFactoryState.overrides, writerFactoryOverride{
		id:      id,
		factory: factory,
	})
	writerFactoryState.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			writerFactoryState.Lock()
			defer writerFactoryState.Unlock()
			for index, override := range writerFactoryState.overrides {
				if override.id != id {
					continue
				}
				last := len(writerFactoryState.overrides) - 1
				copy(writerFactoryState.overrides[index:], writerFactoryState.overrides[index+1:])
				writerFactoryState.overrides[last] = writerFactoryOverride{}
				writerFactoryState.overrides = writerFactoryState.overrides[:last]
				return
			}
		})
	}
}
