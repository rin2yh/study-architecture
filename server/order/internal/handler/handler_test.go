package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newServer は checkout 以外の handler 検証用。外部ポートは成功スタブで固定する。
func newServer(repo repository.OrderRepository) http.Handler {
	return newCheckoutServer(repo, stub.Product{}, stub.Payment{})
}

func newCheckoutServer(repo repository.OrderRepository, product gateway.ProductPort, payment gateway.PaymentPort) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	api.RegisterHandlers(engine, handler.New(repo, product, payment))
	return engine
}

func TestGetHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(stub.Repo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

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
