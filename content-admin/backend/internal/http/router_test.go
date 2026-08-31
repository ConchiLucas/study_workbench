package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-content-admin/internal/catalog"
	"github.com/conchi/study-content-admin/internal/configclient"
	httpapi "github.com/conchi/study-content-admin/internal/http"
)

func TestHealthz(t *testing.T) {
	r := httpapi.NewRouter(httpapi.Deps{Catalog: catalog.NewService(configclient.New(""))})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCatalogUnavailable(t *testing.T) {
	r := httpapi.NewRouter(httpapi.Deps{Catalog: catalog.NewService(configclient.New(""))})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/runtime-config/catalog", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "无法加载共享配置中心", body["error"])
}

func TestCatalogOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/runtime/v1/configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"schemaVersion": "1",
				"generatedAt":   time.Now().UTC().Format(time.RFC3339),
				"ai": map[string]any{
					"activeProviderId": "p1",
					"providers": []map[string]any{{
						"id": "p1", "label": "Demo", "type": "openai-compatible",
						"baseUrl": "http://x", "apiKey": "k", "model": "m",
						"maxTokens": 100, "capabilities": []string{"TEXT_GENERATION"},
						"options": map[string]string{}, "enabled": true,
					}},
				},
				"databases":     []any{},
				"objectStorage": map[string]any{"configured": false},
				"localCli":      map[string]any{"activeConfigId": "", "configs": []any{}},
			})
		case "/api/admin/v1/configuration/image-models", "/api/admin/v1/configuration/video-models", "/api/admin/v1/configuration/voice-models":
			_ = json.NewEncoder(w).Encode(map[string]any{"activeProviderId": "", "providers": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := configclient.New(srv.URL)
	r := httpapi.NewRouter(httpapi.Deps{Catalog: catalog.NewService(client)})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/runtime-config/catalog", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body catalog.Catalog
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "p1", body.AI.Active)
	require.Len(t, body.AI.Providers, 1)
	require.True(t, body.AI.Providers[0].Active)
}
