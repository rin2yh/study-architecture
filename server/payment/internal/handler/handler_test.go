package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-service-base-architecture/server/payment/api"
	"github.com/rin2yh/study-service-base-architecture/server/payment/internal/db"
)

type stubRepo struct {
	payments []db.PaymentPayment
	err      error
}

func (s stubRepo) ListPayments(context.Context) ([]db.PaymentPayment, error) {
	return s.payments, s.err
}

func newServer(repo stubRepo) http.Handler {
	si := api.NewStrictHandler(New(repo), nil)
	return api.HandlerFromMux(si, http.NewServeMux())
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

func TestListPayments(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := stubRepo{payments: []db.PaymentPayment{
		{ID: 1, OrderID: 10, AmountCents: 1980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].OrderId != 10 || got[0].AmountCents != 1980 || got[0].Method != "card" || got[0].Status != "paid" || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected payment: %+v", got[0])
	}
}

func TestListPaymentsError(t *testing.T) {
	repo := stubRepo{err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
