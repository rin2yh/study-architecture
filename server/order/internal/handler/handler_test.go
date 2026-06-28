package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newServer(h *handler.Handler) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	api.RegisterHandlers(engine, h)
	return engine
}

func TestGetHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(handler.New(nil, nil, nil, nil, nil, nil)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body api.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, want ok", body.Status)
	}
}
