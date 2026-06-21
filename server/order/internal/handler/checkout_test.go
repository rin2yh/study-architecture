package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func newCheckoutServer(command handler.Command, product gateway.ProductPort, payment gateway.PaymentPort) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	api.RegisterHandlers(engine, handler.New(nil, command, product, payment))
	return engine
}

type checkoutRecorder struct {
	stub.OrderStub
	err      error
	gotTotal int64
	gotLines []rdb.CheckoutLine
}

func (r *checkoutRecorder) Checkout(_ context.Context, memberID int64, status string, total int64, lines []rdb.CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error) {
	r.gotTotal = total
	r.gotLines = lines
	if r.err != nil {
		return db.OrderOrder{}, nil, r.err
	}
	order := db.OrderOrder{ID: 7, MemberID: memberID, Status: status, TotalCents: total}
	items := make([]db.OrderOrderItem, 0, len(lines))
	for i, l := range lines {
		items = append(items, db.OrderOrderItem{
			ID: int64(i + 1), OrderID: order.ID, ProductID: l.ProductID,
			ProductName: l.ProductName, UnitPriceCents: l.UnitPriceCents, Quantity: l.Quantity,
		})
	}
	return order, items, nil
}

func twoProducts() stub.Product {
	return stub.Product{Snapshots: map[int64]gateway.ProductSnapshot{
		100: {ID: 100, Name: "Widget", UnitPriceCents: 500},
		200: {ID: 200, Name: "Gadget", UnitPriceCents: 1500},
	}}
}

func postCheckout(command handler.Command, product gateway.ProductPort, payment gateway.PaymentPort, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	newCheckoutServer(command, product, payment).ServeHTTP(rec, req)
	return rec
}

func TestCheckout(t *testing.T) {
	const valid = `{"memberId":20,"paymentMethod":"card","items":[{"productId":100,"quantity":2}]}`
	type args struct {
		checkoutErr error
		product     gateway.ProductPort
		payment     gateway.PaymentPort
		body        string
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
		{"正常系 確定し 201", args{nil, twoProducts(), stub.Payment{ID: 1}, valid}, want{http.StatusCreated, ""}},
		{"準正常系 明細が空配列は 400 bad_request", args{nil, twoProducts(), stub.Payment{}, `{"memberId":20,"paymentMethod":"card","items":[]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 items 欠落は 400 bad_request", args{nil, twoProducts(), stub.Payment{}, `{"memberId":20,"paymentMethod":"card"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 quantity 0 は 400 bad_request", args{nil, twoProducts(), stub.Payment{}, `{"memberId":20,"paymentMethod":"card","items":[{"productId":100,"quantity":0}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 memberId 欠落は 400 bad_request", args{nil, twoProducts(), stub.Payment{}, `{"paymentMethod":"card","items":[{"productId":100,"quantity":1}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 paymentMethod 欠落は 400 bad_request", args{nil, twoProducts(), stub.Payment{}, `{"memberId":20,"items":[{"productId":100,"quantity":1}]}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在 product は 422 unprocessable_entity", args{nil, stub.Product{Err: gateway.ErrProductNotFound}, stub.Payment{}, valid}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 product 呼び出し失敗は 502 bad_gateway", args{nil, stub.Product{Err: errors.New("boom")}, stub.Payment{}, valid}, want{http.StatusBadGateway, "bad_gateway"}},
		{"異常系 注文書き込み失敗は 500 internal", args{errors.New("db failure"), twoProducts(), stub.Payment{}, valid}, want{http.StatusInternalServerError, "internal"}},
		{"異常系 payment 失敗は 502 bad_gateway", args{nil, twoProducts(), stub.Payment{Err: errors.New("boom")}, valid}, want{http.StatusBadGateway, "bad_gateway"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := &checkoutRecorder{err: tt.args.checkoutErr}
			rec := postCheckout(command, tt.args.product, tt.args.payment, tt.args.body)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
			}
		})
	}
}

func TestCheckoutSnapshotAndTotal(t *testing.T) {
	command := &checkoutRecorder{}
	body := `{"memberId":20,"paymentMethod":"card","items":[{"productId":100,"quantity":2},{"productId":200,"quantity":1}]}`
	rec := postCheckout(command, twoProducts(), stub.Payment{ID: 1}, body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	// 500*2 + 1500*1
	if command.gotTotal != 2500 {
		t.Fatalf("total = %d, want 2500", command.gotTotal)
	}
	want := []rdb.CheckoutLine{
		{ProductID: 100, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2},
		{ProductID: 200, ProductName: "Gadget", UnitPriceCents: 1500, Quantity: 1},
	}
	if len(command.gotLines) != len(want) {
		t.Fatalf("lines = %+v, want %+v", command.gotLines, want)
	}
	for i := range want {
		if command.gotLines[i] != want[i] {
			t.Fatalf("line[%d] = %+v, want %+v", i, command.gotLines[i], want[i])
		}
	}

	var got api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.TotalCents != 2500 || got.Status != "confirmed" || got.Items == nil || len(*got.Items) != 2 {
		t.Fatalf("unexpected order: %+v", got)
	}
}
