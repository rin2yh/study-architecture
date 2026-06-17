package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-service-base-architecture/services/shipping/api"
	"github.com/rin2yh/study-service-base-architecture/services/shipping/internal/db"
)

// stubRepo は DB なしで handler を検証するための ShipmentRepository スタブ。
type stubRepo struct {
	shipments []db.ShippingShipment
	err       error
}

func (s stubRepo) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return s.shipments, s.err
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

func TestListShipments(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := stubRepo{shipments: []db.ShippingShipment{
		{ID: 1, OrderID: 100, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/shipments", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Shipment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].TrackingNo != "TRK-1" || got[0].OrderId != 100 || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected shipment: %+v", got[0])
	}
}
