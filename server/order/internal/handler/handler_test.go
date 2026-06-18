package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/internal/testdb"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newServer(repo repository.OrderRepository) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	api.RegisterHandlers(engine, handler.New(repo))
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

func TestListOrders(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := stub.Repo{Orders: []db.OrderOrder{
		{ID: 1, MemberID: 10, Status: "paid", TotalCents: 5000, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].MemberId != 10 || got[0].Status != "paid" || got[0].TotalCents != 5000 || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected order: %+v", got[0])
	}
}

func TestListOrdersError(t *testing.T) {
	repo := stub.Repo{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "internal" {
		t.Fatalf("code = %q, want internal", body.Code)
	}
	if body.Message == "db failure" {
		t.Fatalf("message must not expose internal error: %q", body.Message)
	}
}

// 結合: HTTP → handler → repository → 実 DB を通して 200 と JSON が返ることを確認する
// (handler は presentation 層なので実 DB を通した経路でも検証する)。skip 条件は testdb 参照。
func TestListOrdersWithDB(t *testing.T) {
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
		int64(10), "paid", int64(5000)); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newServer(repository.NewRepository(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].MemberId != 10 || got[0].Status != "paid" || got[0].TotalCents != 5000 {
		t.Fatalf("unexpected order: %+v", got)
	}
}
