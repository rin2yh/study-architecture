package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/stub"
)

func TestCreateShipment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	created := db.ShippingShipment{ID: 10, OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		fake stub.ShipmentStub
		body string
	}
	type want struct {
		status int
		code   string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 配送を作成し 201", args{stub.ShipmentStub{Shipment: created}, `{"orderId":200,"carrier":"佐川急便","trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusCreated, ""}},
		{"異常系 carrier 欠落は 400 bad_request", args{stub.ShipmentStub{}, `{"orderId":200,"trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 orderId 不正は 400 bad_request", args{stub.ShipmentStub{}, `{"orderId":0,"carrier":"佐川急便","trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 DB エラーは 500 internal", args{stub.ShipmentStub{Err: errors.New("db failure")}, `{"orderId":200,"carrier":"佐川急便","trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/shipments", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.fake, tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Shipment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 10 || got.TrackingNo != "TRK-10" {
				t.Fatalf("unexpected shipment: %+v", got)
			}
		})
	}
}

func TestUpdateShipment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := db.ShippingShipment{ID: 1, OrderID: 200, Carrier: "ヤマト運輸", TrackingNo: "TRK-99", Status: "delivered", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		fake stub.ShipmentStub
		path string
		body string
	}
	type want struct {
		status int
		code   string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 配送を更新し 200", args{stub.ShipmentStub{Shipment: updated}, "/shipments/1", `{"status":"delivered"}`}, want{http.StatusOK, ""}},
		{"異常系 status 欠落は 400 bad_request", args{stub.ShipmentStub{}, "/shipments/1", `{}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.ShipmentStub{Err: dberr.ErrNotFound}, "/shipments/99", `{"status":"delivered"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.ShipmentStub{Err: errors.New("db failure")}, "/shipments/1", `{"status":"delivered"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, tt.args.path, bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.fake, tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Shipment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Status != "delivered" {
				t.Fatalf("unexpected shipment: %+v", got)
			}
		})
	}
}
