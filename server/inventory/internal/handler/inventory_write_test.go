package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	"github.com/rin2yh/study-architecture/server/inventory/api"
	"github.com/rin2yh/study-architecture/server/inventory/internal/db"
	"github.com/rin2yh/study-architecture/server/inventory/internal/handler"
	"github.com/rin2yh/study-architecture/server/inventory/internal/rdb"
	"github.com/rin2yh/study-architecture/server/inventory/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command))
}

func post(command handler.Command, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(command).ServeHTTP(rec, req)
	return rec
}

func TestStockIn(t *testing.T) {
	fake := stub.InventoryStub{StockInRow: db.InventoryStockIn{ID: 1, ProductID: 100, Quantity: 50}}
	rec := post(fake, "/stock-ins", `{"productId":100,"quantity":50}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.StockIn
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assert.DeepEqual(t, api.StockIn{Id: 1, ProductId: 100, Quantity: 50}, got)
}

func TestStockInError(t *testing.T) {
	type args struct {
		fake stub.InventoryStub
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
		{"準正常系 productId 欠落は 400 bad_request", args{stub.InventoryStub{}, `{"quantity":5}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 quantity 0 は 400 bad_request", args{stub.InventoryStub{}, `{"productId":1,"quantity":0}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 DB エラーは 500 internal", args{stub.InventoryStub{Err: errors.New("db failure")}, `{"productId":1,"quantity":5}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(tt.args.fake, "/stock-ins", tt.args.body)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestReserve(t *testing.T) {
	rec := post(stub.InventoryStub{}, "/reservations", `{"orderId":5,"lines":[{"productId":100,"quantity":2}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.ReservationResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.OrderId != 5 {
		t.Fatalf("orderId = %d, want 5", got.OrderId)
	}
}

func TestReserveError(t *testing.T) {
	type args struct {
		fake stub.InventoryStub
		body string
	}
	type want struct {
		status int
		code   string
	}
	const valid = `{"orderId":5,"lines":[{"productId":100,"quantity":2}]}`
	tests := []struct {
		name string
		args args
		want want
	}{
		{"準正常系 lines 空は 400 bad_request", args{stub.InventoryStub{}, `{"orderId":5,"lines":[]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 orderId 欠落は 400 bad_request", args{stub.InventoryStub{}, `{"lines":[{"productId":1,"quantity":1}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 quantity 0 は 400 bad_request", args{stub.InventoryStub{}, `{"orderId":5,"lines":[{"productId":1,"quantity":0}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 在庫不足は 409 conflict", args{stub.InventoryStub{ReserveErr: rdb.ErrInsufficientStock}, valid}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.InventoryStub{ReserveErr: errors.New("db failure")}, valid}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(tt.args.fake, "/reservations", tt.args.body)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestReleaseReservation(t *testing.T) {
	t.Run("正常系 解放は 204 no content", func(t *testing.T) {
		rec := post(stub.InventoryStub{}, "/reservations/5/release", "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("異常系 DB エラーは 500 internal", func(t *testing.T) {
		rec := post(stub.InventoryStub{Err: errors.New("db failure")}, "/reservations/5/release", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500 (body: %s)", rec.Code, rec.Body.String())
		}
		assert.ErrorCode(t, rec.Body.Bytes(), "internal")
	})
}
