package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/db"
	apphttp "github.com/conchi/study-workbench/internal/http"
	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/seed"
	"github.com/conchi/study-workbench/internal/service"
)

func TestHealthz(t *testing.T) {
	r := apphttp.NewRouter(apphttp.Deps{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":{"status":"ok"},"error":null}`, w.Body.String())
}

func TestPostAttempts(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))

	r := repo.New(gdb)
	cfg := mastery.DefaultConfig()
	router := apphttp.NewRouter(apphttp.Deps{
		Attempt: service.NewAttemptService(r, cfg),
	})

	var kpID int64
	require.NoError(t, gdb.Raw(`SELECT id FROM knowledge_points ORDER BY id LIMIT 1`).Scan(&kpID).Error)

	body, _ := json.Marshal(map[string]any{
		"attempts": []map[string]any{
			{"client_id": "x1", "kp_id": kpID, "is_correct": true, "cost_ms": 1200, "source": "quiz"},
		},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/children/1/attempts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data struct {
			States []service.StateDTO `json:"states"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.States, 1)
	require.Equal(t, "learning", resp.Data.States[0].Status)
}
