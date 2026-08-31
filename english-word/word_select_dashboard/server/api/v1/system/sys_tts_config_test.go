package system

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conchi/go-react-template/server/global"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openTTSConfigAPITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&sysModel.TTSProviderConfig{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func TestTTSConfigAPIGetMasksAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTTSConfigAPITestDB(t)
	row := sysModel.TTSProviderConfig{
		ProviderID: "xiaomi-mimo-tts",
		Label:      "Xiaomi MiMo TTS",
		Type:       "mimo-tts",
		BaseURL:    "https://api.xiaomimimo.com/v1",
		ApiKey:     "stored-secret",
		Model:      "mimo-v2.5-tts",
		Voice:      "Chloe",
		Enabled:    true,
		Active:     true,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/tts/config", nil)
	new(TTSConfigApi).GetConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if strings.Contains(body, "stored-secret") {
		t.Fatalf("response exposed API Key: %s", body)
	}
	if !strings.Contains(body, `"api_key":""`) || !strings.Contains(body, `"api_key_configured":true`) {
		t.Fatalf("response did not contain masked configured key: %s", body)
	}
}

func TestTTSConfigAPISaveBlankKeyPreservesStoredSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTTSConfigAPITestDB(t)
	row := sysModel.TTSProviderConfig{
		ProviderID: "xiaomi-mimo-tts",
		Label:      "Old",
		Type:       "mimo-tts",
		BaseURL:    "https://api.xiaomimimo.com/v1",
		ApiKey:     "stored-secret",
		Model:      "old-model",
		Voice:      "OldVoice",
		Enabled:    true,
		Active:     true,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	body := bytes.NewBufferString(`{
		"active":"xiaomi-mimo-tts",
		"providers":[{
			"id":"xiaomi-mimo-tts",
			"label":"Xiaomi MiMo TTS",
			"type":"mimo-tts",
			"base_url":"https://api.xiaomimimo.com/v1",
			"api_key":"",
			"model":"mimo-v2.5-tts",
			"voice":"Chloe",
			"enabled":true
		}]
	}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/tts/config", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	new(TTSConfigApi).SaveConfig(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "stored-secret") {
		t.Fatalf("response exposed API Key: %s", recorder.Body.String())
	}
	var persisted sysModel.TTSProviderConfig
	if err := db.First(&persisted, "provider_id = ?", "xiaomi-mimo-tts").Error; err != nil {
		t.Fatalf("load persisted row: %v", err)
	}
	if persisted.ApiKey != "stored-secret" {
		t.Fatalf("expected stored key to be preserved, got %q", persisted.ApiKey)
	}
}

func TestTTSConfigAPIReturnsSafeValidationGuidance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTTSConfigAPITestDB(t)
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	body := bytes.NewBufferString(`{
		"active":"xiaomi-mimo-tts",
		"providers":[{
			"id":"xiaomi-mimo-tts",
			"label":"Xiaomi MiMo TTS",
			"type":"mimo-tts",
			"base_url":"https://attacker.example/v1",
			"api_key":"do-not-expose",
			"model":"mimo-v2.5-tts",
			"voice":"Chloe",
			"enabled":true
		}]
	}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/tts/config", body)
	ctx.Request.Header.Set("Content-Type", "application/json")
	new(TTSConfigApi).SaveConfig(ctx)

	responseBody := recorder.Body.String()
	if !strings.Contains(responseBody, "官方") {
		t.Fatalf("expected safe validation guidance, got %s", responseBody)
	}
	if strings.Contains(responseBody, "do-not-expose") {
		t.Fatalf("validation response exposed secret: %s", responseBody)
	}
}
