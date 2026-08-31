package system_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/initialize"
	sysModel "github.com/conchi/go-react-template/server/model/system"
	systemReq "github.com/conchi/go-react-template/server/model/system/request"
	systemRouter "github.com/conchi/go-react-template/server/router/system"
	"github.com/conchi/go-react-template/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestExecutionConfigRouterRegistersPrivateGetAndPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	publicGroup := engine.Group("/api")
	publicGroup.Use(routeGroupMarker("public"))
	privateGroup := engine.Group("/api")
	privateGroup.Use(routeGroupMarker("private"))

	new(systemRouter.AIConfigRouter).InitAIConfigRouter(publicGroup, privateGroup)
	new(systemRouter.ExecutionConfigRouter).InitExecutionConfigRouter(publicGroup, privateGroup)

	routes := engine.Routes()
	found := make(map[string]bool, len(routes))
	for _, route := range routes {
		found[route.Method+" "+route.Path] = true
	}
	for _, expected := range []string{
		"GET /api/ai/execution-config",
		"POST /api/ai/execution-config",
		"GET /api/ai/config",
		"POST /api/ai/config",
	} {
		if !found[expected] {
			t.Fatalf("missing route %s; got %#v", expected, routes)
		}
	}

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{method: http.MethodGet, path: "/api/ai/execution-config", want: "private"},
		{method: http.MethodPost, path: "/api/ai/execution-config", want: "private"},
		{method: http.MethodGet, path: "/api/ai/config", want: "public"},
		{method: http.MethodPost, path: "/api/ai/config", want: "private"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(`{}`))
			request.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(recorder, request)
			if got := recorder.Header().Get("X-Route-Group"); got != test.want {
				t.Fatalf("route used %q group, want %q; status=%d body=%s", got, test.want, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func routeGroupMarker(value string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Route-Group", value)
		c.Next()
	}
}

func TestInitializedExecutionConfigRouteUsesJWTBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousSystem := global.GVA_CONFIG.System
	previousJWT := global.GVA_CONFIG.JWT
	previousLog := global.GVA_LOG
	previousDB := global.GVA_DB
	global.GVA_CONFIG.System = config.System{RouterPrefix: "/api"}
	global.GVA_CONFIG.JWT = config.JWT{
		SigningKey: "execution-config-test-key", ExpiresTime: "1h", BufferTime: "10m", Issuer: "test",
	}
	db := openInitializedExecutionConfigTestDB(t)
	if err := db.Create(&sysModel.AIProviderConfig{
		ProviderID: "openai", Label: "OpenAI", Type: "openai-compatible",
		BaseURL: "https://api.openai.com/v1", ApiKey: "stored-secret", Model: "gpt-test", MaxTokens: 2048, Active: true,
	}).Error; err != nil {
		t.Fatalf("seed API provider: %v", err)
	}
	if err := db.Create(&sysModel.SentenceExecutorConfig{
		SingletonKey: "default", ExecutorType: "api", ExecutorID: "openai",
	}).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}
	global.GVA_LOG = zap.NewNop()
	global.GVA_DB = db
	t.Cleanup(func() {
		global.GVA_CONFIG.System = previousSystem
		global.GVA_CONFIG.JWT = previousJWT
		global.GVA_LOG = previousLog
		global.GVA_DB = previousDB
	})

	engine := initialize.Routers()
	found := map[string]bool{}
	for _, route := range engine.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, expected := range []string{"GET /api/ai/execution-config", "POST /api/ai/execution-config"} {
		if !found[expected] {
			t.Fatalf("initialized router missing %s", expected)
		}
	}

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/ai/execution-config", bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("private %s without JWT returned %d, want 401; body=%s", method, recorder.Code, recorder.Body.String())
		}
	}

	claims := utils.NewJWT().CreateClaims(systemReq.BaseClaims{ID: 1, Username: "admin"})
	token, err := utils.NewJWT().CreateToken(claims)
	if err != nil {
		t.Fatalf("create test token: %v", err)
	}
	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/ai/execution-config", nil)
	getRequest.Header.Set("x-token", token)
	engine.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK || !bytes.Contains(getRecorder.Body.Bytes(), []byte(`"code":0`)) {
		t.Fatalf("authenticated GET failed: status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}

	postBody := `{
		"active_target":{"type":"api","id":"openai"},
		"api_providers":[{"id":"openai","label":"Updated OpenAI","type":"openai-compatible","base_url":"https://api.openai.com/v1","api_key":"","model":"gpt-updated","max_tokens":4096}],
		"cli_providers":[]
	}`
	postRecorder := httptest.NewRecorder()
	postRequest := httptest.NewRequest(http.MethodPost, "/api/ai/execution-config", bytes.NewBufferString(postBody))
	postRequest.Header.Set("Content-Type", "application/json")
	postRequest.Header.Set("x-token", token)
	engine.ServeHTTP(postRecorder, postRequest)
	if postRecorder.Code != http.StatusOK || !bytes.Contains(postRecorder.Body.Bytes(), []byte(`"code":0`)) {
		t.Fatalf("authenticated POST failed: status=%d body=%s", postRecorder.Code, postRecorder.Body.String())
	}
	var provider sysModel.AIProviderConfig
	if err := db.First(&provider, "provider_id = ?", "openai").Error; err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if provider.ApiKey != "stored-secret" || provider.Model != "gpt-updated" {
		t.Fatalf("authenticated POST persisted wrong provider: %#v", provider)
	}
}

func openInitializedExecutionConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&sysModel.AIProviderConfig{},
		&sysModel.CLIProviderConfig{},
		&sysModel.SentenceExecutorConfig{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}
