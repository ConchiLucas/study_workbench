package system

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/conchi/go-react-template/server/config"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/conchi/go-react-template/server/model/system/request"
	"github.com/conchi/go-react-template/server/utils/gormsafe"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type runtimeInitCaptureWriter struct {
	mu     sync.Mutex
	output strings.Builder
}

func (w *runtimeInitCaptureWriter) Printf(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = fmt.Fprintf(&w.output, format, args...)
}

func (w *runtimeInitCaptureWriter) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.output.Reset()
}

func (w *runtimeInitCaptureWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.output.String()
}

func TestSqliteInitHandlerEnsureDBUsesSafeLogger(t *testing.T) {
	const sentinel = "runtime-initdb-api-key-sentinel"
	writer := new(runtimeInitCaptureWriter)
	restore := gormsafe.SetWriterFactoryForTesting(func(config.GeneralDB) logger.Writer {
		return writer
	})
	t.Cleanup(restore)

	ctx := context.WithValue(context.Background(), "dbtype", "sqlite")
	next, err := NewSqliteInitHandler().EnsureDB(ctx, &request.InitDB{
		DBType: "sqlite",
		DBPath: t.TempDir(),
		DBName: "runtime-initdb",
	})
	if err != nil {
		t.Fatalf("ensure SQLite database: %v", err)
	}
	db, ok := next.Value("db").(*gorm.DB)
	if !ok || db == nil {
		t.Fatalf("EnsureDB did not return a GORM database: %#v", next.Value("db"))
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying runtime database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close runtime database: %v", err)
		}
	})

	filter, ok := db.Logger.(gorm.ParamsFilter)
	if !ok {
		t.Fatalf("runtime database logger does not implement parameter filtering: %T", db.Logger)
	}
	_, params := filter.ParamsFilter(context.Background(), "INSERT INTO ai_provider_configs (api_key) VALUES (?)", sentinel)
	if len(params) != 0 {
		t.Fatalf("runtime database logger retained secret parameters: %#v", params)
	}

	if err := db.AutoMigrate(&sysModel.AIProviderConfig{}); err != nil {
		t.Fatalf("migrate API provider table: %v", err)
	}
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "duplicate", Type: "openai-compatible", BaseURL: "https://example.com/v1",
		ApiKey: "existing-key", Model: "existing-model",
	}).Error; err != nil {
		t.Fatalf("seed duplicate provider: %v", err)
	}

	writer.Reset()
	result := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "duplicate", Type: "openai-compatible", BaseURL: "https://example.com/v1",
		ApiKey: sentinel, Model: "new-model",
	})
	if result.Error == nil {
		t.Fatal("expected duplicate provider write to fail")
	}
	output := writer.String()
	if strings.Contains(output, sentinel) {
		t.Fatalf("runtime database log exposed API key: %s", output)
	}
	if !strings.Contains(output, "VALUES (") || !strings.Contains(output, "?") {
		t.Fatalf("runtime database log did not retain SQL placeholders: %s", output)
	}
}
