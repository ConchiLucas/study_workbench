package configclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/stretchr/testify/require"
)

func TestRefreshStoresSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/runtime/v1/configuration", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schemaVersion": "1",
			"generatedAt":   time.Now().UTC().Format(time.RFC3339),
			"ai":            map[string]any{"activeProviderId": "p1", "providers": []any{}},
			"databases":     []any{},
			"objectStorage": map[string]any{"configured": false},
			"localCli":      map[string]any{"activeConfigId": "", "configs": []any{}},
		})
	}))
	defer srv.Close()

	client := configclient.New(srv.URL)
	value, err := client.Refresh(context.Background())
	require.NoError(t, err)
	require.Equal(t, "1", value.SchemaVersion)

	current, err := client.Current()
	require.NoError(t, err)
	require.Equal(t, "1", current.SchemaVersion)
}

func TestRefreshRejectsBadSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"schemaVersion": "2"})
	}))
	defer srv.Close()

	_, err := configclient.New(srv.URL).Refresh(context.Background())
	require.ErrorIs(t, err, configclient.ErrUnavailable)
}

func TestCurrentBeforeLoad(t *testing.T) {
	_, err := configclient.New("http://example.invalid").Current()
	require.ErrorIs(t, err, configclient.ErrNotLoaded)
}
