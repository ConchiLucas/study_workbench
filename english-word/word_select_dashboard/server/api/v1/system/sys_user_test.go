package system

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conchi/go-react-template/server/global"
	"github.com/gin-gonic/gin"
)

func TestLoginRequiresInitializedDatabase(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	global.GVA_DB = nil

	body, err := json.Marshal(map[string]string{
		"username": "admin",
		"password": "123456",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/base/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req

	new(BaseApi).Login(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("数据库尚未初始化")) {
		t.Fatalf("expected initialization guidance, got %s", rec.Body.String())
	}
}
