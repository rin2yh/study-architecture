package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command, nil, nil))
}

func newCheckoutServer(command handler.Command, product gateway.ProductPort, payment gateway.PaymentPort) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	api.RegisterHandlers(engine, handler.New(nil, command, product, payment))
	return engine
}

func postCheckout(command handler.Command, product gateway.ProductPort, payment gateway.PaymentPort, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	newCheckoutServer(command, product, payment).ServeHTTP(rec, req)
	return rec
}

func TestCreateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	if _, err := pool.Exec(t.Context(), `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(`{"memberId":20,"status":"pending","totalCents":1980}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewOrderCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Id == 0 || got.MemberId != 20 || got.Status != "pending" || got.TotalCents != 1980 {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestCreateOrderError(t *testing.T) {
	type args struct {
		fake stub.OrderStub
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
		{"異常系 status 欠落は 400 bad_request", args{stub.OrderStub{}, `{"memberId":20,"totalCents":1980}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 memberId 欠落は 400 bad_request", args{stub.OrderStub{}, `{"status":"pending","totalCents":1980}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 totalCents 負値は 422 unprocessable_entity", args{stub.OrderStub{}, `{"memberId":20,"status":"pending","totalCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 DB エラーは 500 internal", args{stub.OrderStub{Err: errors.New("db failure")}, `{"memberId":20,"status":"pending","totalCents":1980}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestUpdateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
		int64(10), "pending", int64(1980)); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/orders/1", bytes.NewReader([]byte(`{"status":"paid"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewOrderCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Id != 1 || got.Status != "paid" || got.MemberId != 10 || got.TotalCents != 1980 {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestUpdateOrderError(t *testing.T) {
	type args struct {
		fake stub.OrderStub
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
		{"異常系 status 欠落は 400 bad_request", args{stub.OrderStub{}, "/orders/1", `{}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.OrderStub{Err: dberr.ErrNotFound}, "/orders/99", `{"status":"paid"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.OrderStub{Err: errors.New("db failure")}, "/orders/1", `{"status":"paid"}`}, want{http.StatusInternalServerError, "internal"}},
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
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestCheckout(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	if _, err := pool.Exec(t.Context(), `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rec := postCheckout(rdb.NewOrderCommand(pool), stub.TwoProducts(), stub.Payment{ID: 1},
		`{"memberId":20,"paymentMethod":"card","items":[{"productId":100,"quantity":2},{"productId":200,"quantity":1}]}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// 500*2 + 1500*1 を商品スナップショットから合算した結果が永続化される
	if got.TotalCents != 2500 || got.Status != "confirmed" || got.Items == nil || len(*got.Items) != 2 {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestCheckoutError(t *testing.T) {
	const valid = `{"memberId":20,"paymentMethod":"card","items":[{"productId":100,"quantity":2}]}`
	type args struct {
		command handler.Command
		product gateway.ProductPort
		payment gateway.PaymentPort
		body    string
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
		{"異常系 明細が空配列は 400 bad_request", args{stub.OrderStub{}, stub.TwoProducts(), stub.Payment{}, `{"memberId":20,"paymentMethod":"card","items":[]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 items 欠落は 400 bad_request", args{stub.OrderStub{}, stub.TwoProducts(), stub.Payment{}, `{"memberId":20,"paymentMethod":"card"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 quantity 0 は 400 bad_request", args{stub.OrderStub{}, stub.TwoProducts(), stub.Payment{}, `{"memberId":20,"paymentMethod":"card","items":[{"productId":100,"quantity":0}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 memberId 欠落は 400 bad_request", args{stub.OrderStub{}, stub.TwoProducts(), stub.Payment{}, `{"paymentMethod":"card","items":[{"productId":100,"quantity":1}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 paymentMethod 欠落は 400 bad_request", args{stub.OrderStub{}, stub.TwoProducts(), stub.Payment{}, `{"memberId":20,"items":[{"productId":100,"quantity":1}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在 product は 422 unprocessable_entity", args{stub.OrderStub{}, stub.Product{Err: gateway.ErrProductNotFound}, stub.Payment{}, valid}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 product 呼び出し失敗は 502 bad_gateway", args{stub.OrderStub{}, stub.Product{Err: errors.New("boom")}, stub.Payment{}, valid}, want{http.StatusBadGateway, "bad_gateway"}},
		{"異常系 注文書き込み失敗は 500 internal", args{stub.OrderStub{Err: errors.New("db failure")}, stub.TwoProducts(), stub.Payment{}, valid}, want{http.StatusInternalServerError, "internal"}},
		{"異常系 payment 失敗は 502 bad_gateway", args{stub.OrderStub{Order: db.OrderOrder{ID: 7}}, stub.TwoProducts(), stub.Payment{Err: errors.New("boom")}, valid}, want{http.StatusBadGateway, "bad_gateway"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postCheckout(tt.args.command, tt.args.product, tt.args.payment, tt.args.body)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
