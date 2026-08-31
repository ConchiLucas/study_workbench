package gormsafe

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/conchi/go-react-template/server/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type loggerSafetyRecord struct {
	ID     uint   `gorm:"primarykey"`
	Name   string `gorm:"uniqueIndex"`
	Secret string
}

func TestWriterFactoryOverridesRestoreSafelyOutOfOrder(t *testing.T) {
	var callsA atomic.Int32
	var callsB atomic.Int32
	discard := log.New(io.Discard, "", 0)

	restoreA := SetWriterFactoryForTesting(func(config.GeneralDB) logger.Writer {
		callsA.Add(1)
		return discard
	})
	_ = Config(config.GeneralDB{})
	restoreB := SetWriterFactoryForTesting(func(config.GeneralDB) logger.Writer {
		callsB.Add(1)
		return discard
	})
	_ = Config(config.GeneralDB{})

	restoreA()
	_ = Config(config.GeneralDB{})
	restoreB()
	restoreA()
	restoreB()
	_ = Config(config.GeneralDB{})

	if got := callsA.Load(); got != 1 {
		t.Fatalf("factory A remained active after its cleanup: %d calls", got)
	}
	if got := callsB.Load(); got != 2 {
		t.Fatalf("factory B was lost before its cleanup or remained after it: %d calls", got)
	}
}

type captureWriter struct {
	mu     sync.Mutex
	output strings.Builder
}

func (w *captureWriter) Printf(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = fmt.Fprintf(&w.output, format, args...)
}

func (w *captureWriter) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.output.Reset()
}

func (w *captureWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.output.String()
}

func TestConfigWithWriterNeverInterpolatesTraceParameters(t *testing.T) {
	const (
		infoSentinel  = "shared-info-secret-sentinel"
		errorSentinel = "shared-error-secret-sentinel"
	)
	writer := new(captureWriter)
	general := config.GeneralDB{
		Prefix:   "safe_",
		Singular: true,
		LogMode:  "info",
	}
	gormConfig := ConfigWithWriter(general, writer)
	if !gormConfig.DisableForeignKeyConstraintWhenMigrating {
		t.Fatal("safe config lost migration constraint setting")
	}
	if got := gormConfig.NamingStrategy.TableName("LoggerSafetyRecord"); got != "safe_logger_safety_record" {
		t.Fatalf("safe config lost naming strategy: %q", got)
	}

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), gormConfig)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying test database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if err := db.AutoMigrate(&loggerSafetyRecord{}); err != nil {
		t.Fatalf("migrate logger record: %v", err)
	}

	writer.Reset()
	if err := db.Create(&loggerSafetyRecord{Name: "duplicate", Secret: infoSentinel}).Error; err != nil {
		t.Fatalf("create info record: %v", err)
	}
	assertParameterizedLog(t, writer.String(), infoSentinel, "info")

	writer.Reset()
	result := db.Create(&loggerSafetyRecord{Name: "duplicate", Secret: errorSentinel})
	if result.Error == nil {
		t.Fatal("expected duplicate record write to fail")
	}
	assertParameterizedLog(t, writer.String(), errorSentinel, "error")
}

func assertParameterizedLog(t *testing.T, output, sentinel, traceKind string) {
	t.Helper()
	if strings.Contains(output, sentinel) {
		t.Fatalf("%s trace interpolated secret parameter: %s", traceKind, output)
	}
	if !strings.Contains(output, "VALUES (") || !strings.Contains(output, "?") {
		t.Fatalf("%s trace did not retain SQL placeholders: %s", traceKind, output)
	}
}
