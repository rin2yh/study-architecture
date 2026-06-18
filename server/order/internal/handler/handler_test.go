package handler

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

	"github.com/rin2yh/study-architecture/server/internal/httperror"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type stubRepo struct {
	orders []db.OrderOrder
	err    error
}

func (s stubRepo) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.orders, s.err
}

func newServer(repo stubRepo) http.Handler {
	engine := gin.New()
	si := api.NewStrictHandlerWithOptions(New(repo), nil, api.StrictGinServerOptions{
		RequestErrorHandlerFunc:  httperror.BadRequest,
		HandlerErrorFunc:         httperror.Internal,
		ResponseErrorHandlerFunc: httperror.Internal,
	})
	api.RegisterHandlers(engine, si)
	return engine
}

func TestGetHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(stubRepo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

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
	repo := stubRepo{orders: []db.OrderOrder{
		{ID: 1, MemberID: 42, Status: "pending", TotalCents: 1980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
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
	if got[0].MemberId != 42 || got[0].Status != "pending" || got[0].TotalCents != 1980 || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected order: %+v", got[0])
	}
}

func TestListOrdersError(t *testing.T) {
	repo := stubRepo{err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	var body httperror.Response
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
