package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/rdb"
	"github.com/rin2yh/study-architecture/server/shipping/internal/stub"
)

func TestListShipments(t *testing.T) {
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
	newServer(rdb.NewShipmentQuery(pool), rdb.NewShipmentCommand(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/shipments", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Shipment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []api.Shipment{{OrderId: 100, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped"}}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Shipment{}, "Id", "CreatedAt")); diff != "" {
		t.Fatalf("shipments mismatch (-want +got):\n%s", diff)
	}
}

func TestListShipmentsError(t *testing.T) {
	fake := stub.ShipmentStub{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(fake, fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/shipments", nil))

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

func TestGetShipment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	shipment := db.ShippingShipment{ID: 1, OrderID: 100, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		fake stub.ShipmentStub
		path string
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
		{"正常系 配送を返す", args{stub.ShipmentStub{Shipment: shipment}, "/shipments/1"}, want{http.StatusOK, ""}},
		{"異常系 未存在は 404 not_found", args{stub.ShipmentStub{Err: dberr.ErrNotFound}, "/shipments/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.ShipmentStub{Err: errors.New("db failure")}, "/shipments/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newServer(tt.args.fake, tt.args.fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Shipment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.TrackingNo != "TRK-1" {
				t.Fatalf("unexpected shipment: %+v", got)
			}
		})
	}
}
