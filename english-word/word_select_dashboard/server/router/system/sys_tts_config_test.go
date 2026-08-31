package system

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/middleware"
	systemReq "github.com/conchi/go-react-template/server/model/system/request"
	"github.com/conchi/go-react-template/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestTTSConfigRouterRegistersDedicatedEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	new(TTSConfigRouter).InitTTSConfigRouter(engine.Group("/api"), engine.Group("/api"))

	routes := engine.Routes()
	found := map[string]bool{}
	for _, route := range routes {
		found[route.Method+" "+route.Path] = true
	}
	for _, expected := range []string{"GET /api/tts/config", "POST /api/tts/config"} {
		if !found[expected] {
			t.Fatalf("missing route %s; got %#v", expected, routes)
		}
	}
}

func TestConfigMutationRoutesRequireJWTAndAcceptValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousJWT := global.GVA_CONFIG.JWT
	previousLog := global.GVA_LOG
	global.GVA_CONFIG.JWT = config.JWT{
		SigningKey:  "test-signing-key",
		ExpiresTime: "1h",
		BufferTime:  "10m",
		Issuer:      "test",
	}
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_CONFIG.JWT = previousJWT
		global.GVA_LOG = previousLog
	})

	engine := gin.New()
	publicGroup := engine.Group("/api")
	privateGroup := engine.Group("/api")
	privateGroup.Use(middleware.JWTAuth())
	new(AIConfigRouter).InitAIConfigRouter(publicGroup, privateGroup)
	new(TTSConfigRouter).InitTTSConfigRouter(publicGroup, privateGroup)

	for _, path := range []string{"/api/ai/config", "/api/tts/config"} {
		unauthorized := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(unauthorized, request)
		if unauthorized.Code != http.StatusUnauthorized {
			t.Fatalf("expected unauthenticated POST %s to return 401, got %d", path, unauthorized.Code)
		}

		claims := utils.NewJWT().CreateClaims(systemReq.BaseClaims{ID: 1, Username: "admin"})
		token, err := utils.NewJWT().CreateToken(claims)
		if err != nil {
			t.Fatalf("create test token: %v", err)
		}
		authorized := httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("x-token", token)
		engine.ServeHTTP(authorized, request)
		if authorized.Code == http.StatusUnauthorized {
			t.Fatalf("expected authenticated POST %s to reach handler", path)
		}
	}
}
