package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/handler"
	"github.com/rin2yh/study-architecture/server/shipping/internal/rdb"
	"github.com/rin2yh/study-architecture/server/shipping/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command))
}

func TestCreateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_OPS")
	if _, err := pool.Exec(t.Context(), `TRUNCATE shipping.shipments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/shipments", bytes.NewReader([]byte(`{"orderId":200,"carrier":"佐川急便","trackingNo":"TRK-10","status":"pending"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewShipmentCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Shipment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Shipment{OrderId: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"}
	assert.Equal(t, want, got, "Id", "CreatedAt")
}

func TestCreateShipmentError(t *testing.T) {
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
		{"異常系 carrier 欠落は 400 bad_request", args{stub.ShipmentStub{}, `{"orderId":200,"trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 orderId 不正は 400 bad_request", args{stub.ShipmentStub{}, `{"orderId":0,"carrier":"佐川急便","trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 DB エラーは 500 internal", args{stub.ShipmentStub{Err: errors.New("db failure")}, `{"orderId":200,"carrier":"佐川急便","trackingNo":"TRK-10","status":"pending"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/shipments", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestUpdateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_OPS")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE shipping.shipments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO shipping.shipments (order_id, carrier, tracking_no, status) VALUES ($1, $2, $3, $4)`,
		int64(100), "ヤマト運輸", "TRK-1", "shipped"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/shipments/1", bytes.NewReader([]byte(`{"status":"delivered"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewShipmentCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Shipment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Shipment{OrderId: 100, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "delivered"}
	assert.Equal(t, want, got, "Id", "CreatedAt")
}

func TestUpdateShipmentError(t *testing.T) {
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
		{"異常系 status 欠落は 400 bad_request", args{stub.ShipmentStub{}, "/shipments/1", `{}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.ShipmentStub{Err: dberr.ErrNotFound}, "/shipments/99", `{"status":"delivered"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.ShipmentStub{Err: errors.New("db failure")}, "/shipments/1", `{"status":"delivered"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, tt.args.path, bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
