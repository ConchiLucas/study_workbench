package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/db"
	apphttp "github.com/conchi/study-workbench/internal/http"
	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/seed"
	"github.com/conchi/study-workbench/internal/service"
)

func newPlanRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))
	_, err = seed.Questions(gdb)
	require.NoError(t, err)

	r := repo.New(gdb)
	attempts := service.NewAttemptService(r, mastery.DefaultConfig())
	return apphttp.NewRouter(apphttp.Deps{
		Attempt: attempts,
		Plan:    service.NewPlanService(r, attempts),
	}), gdb
}

func do(t *testing.T, router *gin.Engine, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	return w.Code, resp
}

func TestPlanRoutesWalkThroughAWholeSession(t *testing.T) {
	router, gdb := newPlanRouter(t)

	code, resp := do(t, router, http.MethodGet, "/api/v1/children/1/plans/today", nil)
	require.Equal(t, http.StatusOK, code, resp)

	data := resp["data"].(map[string]any)
	planID := int64(data["plan"].(map[string]any)["id"].(float64))
	items := data["items"].([]any)
	require.Len(t, items, 10)

	code, _ = do(t, router, http.MethodPost,
		fmt.Sprintf("/api/v1/children/1/plans/%d/start", planID), nil)
	require.Equal(t, http.StatusOK, code)

	for _, raw := range items {
		item := raw.(map[string]any)
		itemID := int64(item["id"].(float64))
		questionID := int64(item["question"].(map[string]any)["id"].(float64))

		var answer string
		require.NoError(t, gdb.Raw(`SELECT answer FROM questions WHERE id = ?`, questionID).
			Scan(&answer).Error)
		var a struct{ Index int }
		require.NoError(t, json.Unmarshal([]byte(answer), &a))

		code, resp := do(t, router, http.MethodPost,
			fmt.Sprintf("/api/v1/children/1/plans/%d/items/%d/answer", planID, itemID),
			map[string]any{"option_index": a.Index, "cost_ms": 3000})
		require.Equal(t, http.StatusOK, code, resp)
		require.True(t, resp["data"].(map[string]any)["correct"].(bool))
	}

	code, resp = do(t, router, http.MethodPost,
		fmt.Sprintf("/api/v1/children/1/plans/%d/finish", planID), nil)
	require.Equal(t, http.StatusOK, code, resp)
	require.EqualValues(t, 3, resp["data"].(map[string]any)["stars"].(float64))
}

func TestUnknownPlanRoutesReturn404(t *testing.T) {
	router, _ := newPlanRouter(t)

	code, resp := do(t, router, http.MethodPost, "/api/v1/children/1/plans/424242/start", nil)
	require.Equal(t, http.StatusNotFound, code, resp)
	require.NotNil(t, resp["error"])

	code, _ = do(t, router, http.MethodPost,
		"/api/v1/children/1/plans/424242/items/1/answer", map[string]any{"option_index": 0})
	require.Equal(t, http.StatusNotFound, code)
}

func TestExtraPlanBeyondDailyCapReturns409(t *testing.T) {
	router, _ := newPlanRouter(t)

	code, _ := do(t, router, http.MethodGet, "/api/v1/children/1/plans/today", nil)
	require.Equal(t, http.StatusOK, code)

	for i := 0; i < 2; i++ {
		code, resp := do(t, router, http.MethodPost, "/api/v1/children/1/plans/extra", nil)
		require.Equal(t, http.StatusOK, code, resp)
	}
	code, resp := do(t, router, http.MethodPost, "/api/v1/children/1/plans/extra", nil)
	require.Equal(t, http.StatusConflict, code, resp)
}

func TestMissingQuestionBankReturns422(t *testing.T) {
	gdb, err := db.OpenMemory()
	require.NoError(t, err)
	require.NoError(t, db.Migrate(gdb))
	require.NoError(t, seed.Catalog(gdb))

	r := repo.New(gdb)
	attempts := service.NewAttemptService(r, mastery.DefaultConfig())
	router := apphttp.NewRouter(apphttp.Deps{Plan: service.NewPlanService(r, attempts)})

	code, resp := do(t, router, http.MethodGet, "/api/v1/children/1/plans/today", nil)
	require.Equal(t, http.StatusUnprocessableEntity, code, resp)
}
