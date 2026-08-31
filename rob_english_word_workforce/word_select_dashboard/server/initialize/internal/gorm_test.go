package internal

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/conchi/go-react-template/server/config"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGormLoggerDoesNotInterpolateSecretParameters(t *testing.T) {
	const sentinel = "gorm-log-secret-sentinel"
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
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

	output, err := captureStdout(func() error {
		secureLogger := Gorm.Config(config.GeneralDB{LogMode: "error"}).Logger
		result := db.Session(&gorm.Session{Logger: secureLogger}).Create(&sysModel.AIProviderConfig{
			ProviderID: "duplicate", Type: "openai-compatible", BaseURL: "https://example.com/v1",
			ApiKey: sentinel, Model: "new-model",
		})
		if result.Error == nil {
			return errors.New("expected duplicate provider write to fail")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("exercise GORM logger: %v", err)
	}
	if strings.Contains(output, sentinel) {
		t.Fatalf("GORM log interpolated secret parameter: %s", output)
	}
	if !strings.Contains(output, "VALUES (") || !strings.Contains(output, "?") {
		t.Fatalf("GORM log did not retain SQL placeholder: %s", output)
	}
}

func captureStdout(run func() error) (string, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	previous := os.Stdout
	os.Stdout = writer
	runErr := run()
	os.Stdout = previous
	closeErr := writer.Close()
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		return string(output), runErr
	}
	if closeErr != nil {
		return string(output), closeErr
	}
	return string(output), readErr
}
