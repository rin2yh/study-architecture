package gateway_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
)

func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestProductClientFetchProduct(t *testing.T) {
	type want struct {
		snap    gateway.ProductSnapshot
		errIs   error
		wantErr bool
	}
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    want
	}{
		{
			"正常系 200 を ProductSnapshot に複写",
			jsonHandler(http.StatusOK, `{"id":100,"sku":"x","name":"Widget","priceCents":500,"createdAt":"2026-01-01T00:00:00Z"}`),
			want{snap: gateway.ProductSnapshot{ID: 100, Name: "Widget", UnitPriceCents: 500}},
		},
		{
			"準正常系 404 は ErrProductNotFound",
			jsonHandler(http.StatusNotFound, `{"code":"not_found","message":"product 100 not found"}`),
			want{errIs: gateway.ErrProductNotFound, wantErr: true},
		},
		{
			"異常系 500 は ErrUpstream",
			jsonHandler(http.StatusInternalServerError, `{"code":"internal","message":"boom"}`),
			want{errIs: gateway.ErrUpstream, wantErr: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			t.Setenv("PRODUCT_API_URL", srv.URL)
			c, err := gateway.NewProductClient()
			if err != nil {
				t.Fatalf("NewProductClient: %v", err)
			}

			got, err := c.FetchProduct(t.Context(), 100)
			if tt.want.wantErr {
				if !errors.Is(err, tt.want.errIs) {
					t.Fatalf("err = %v, want errors.Is %v", err, tt.want.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchProduct: %v", err)
			}
			if got != tt.want.snap {
				t.Fatalf("snap = %+v, want %+v", got, tt.want.snap)
			}
		})
	}
}

func TestProductClientFetchProductTransportError(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, `{}`))
	t.Setenv("PRODUCT_API_URL", srv.URL)
	c, err := gateway.NewProductClient()
	if err != nil {
		t.Fatalf("NewProductClient: %v", err)
	}
	srv.Close() // 接続不能にしてから呼ぶ

	if _, err := c.FetchProduct(t.Context(), 100); !errors.Is(err, gateway.ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
}

func TestMemberClientFetchAddress(t *testing.T) {
	type want struct {
		snap    gateway.AddressSnapshot
		errIs   error
		wantErr bool
	}
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    want
	}{
		{
			"正常系 200 を AddressSnapshot に複写",
			jsonHandler(http.StatusOK, `{"id":5,"memberId":20,"recipient":"山田太郎","postalCode":"1500001","prefecture":"東京都","city":"渋谷区","line1":"神宮前1-2-3","createdAt":"2026-01-01T00:00:00Z"}`),
			want{snap: gateway.AddressSnapshot{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}},
		},
		{
			"準正常系 404 は ErrAddressNotFound",
			jsonHandler(http.StatusNotFound, `{"code":"not_found","message":"address not found"}`),
			want{errIs: gateway.ErrAddressNotFound, wantErr: true},
		},
		{
			"異常系 500 は ErrUpstream",
			jsonHandler(http.StatusInternalServerError, `{"code":"internal","message":"boom"}`),
			want{errIs: gateway.ErrUpstream, wantErr: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			t.Setenv("MEMBER_API_URL", srv.URL)
			c, err := gateway.NewMemberClient()
			if err != nil {
				t.Fatalf("NewMemberClient: %v", err)
			}

			got, err := c.FetchAddress(t.Context(), 20, 5)
			if tt.want.wantErr {
				if !errors.Is(err, tt.want.errIs) {
					t.Fatalf("err = %v, want errors.Is %v", err, tt.want.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchAddress: %v", err)
			}
			if got != tt.want.snap {
				t.Fatalf("snap = %+v, want %+v", got, tt.want.snap)
			}
		})
	}
}

func TestMemberClientFetchAddressTransportError(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, `{}`))
	t.Setenv("MEMBER_API_URL", srv.URL)
	c, err := gateway.NewMemberClient()
	if err != nil {
		t.Fatalf("NewMemberClient: %v", err)
	}
	srv.Close() // 接続不能にしてから呼ぶ

	if _, err := c.FetchAddress(t.Context(), 20, 5); !errors.Is(err, gateway.ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
}

func TestPaymentClientCreatePayment(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantID  int64
		wantErr bool
	}{
		{"正常系 201 で payment id を返す", jsonHandler(http.StatusCreated, `{"id":42,"orderId":7,"amountCents":2500,"method":"card","status":"pending","createdAt":"2026-01-01T00:00:00Z"}`), 42, false},
		{"異常系 500 は ErrUpstream", jsonHandler(http.StatusInternalServerError, `{"code":"internal"}`), 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			t.Setenv("PAYMENT_API_URL", srv.URL)
			c, err := gateway.NewPaymentClient()
			if err != nil {
				t.Fatalf("NewPaymentClient: %v", err)
			}

			id, err := c.CreatePayment(t.Context(), 7, 2500, "card", "idem-key", gateway.AddressSnapshot{Recipient: "山田太郎"})
			if tt.wantErr {
				if !errors.Is(err, gateway.ErrUpstream) {
					t.Fatalf("err = %v, want ErrUpstream", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreatePayment: %v", err)
			}
			if id != tt.wantID {
				t.Fatalf("id = %d, want %d", id, tt.wantID)
			}
		})
	}
}

func TestNewClientRequiresEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		ctor func() error
	}{
		{"異常系 PRODUCT_API_URL 未設定はエラー", "PRODUCT_API_URL", func() error { _, err := gateway.NewProductClient(); return err }},
		{"異常系 PAYMENT_API_URL 未設定はエラー", "PAYMENT_API_URL", func() error { _, err := gateway.NewPaymentClient(); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.env, "")
			if err := tt.ctor(); err == nil {
				t.Fatalf("ctor with empty %s: want error", tt.env)
			}
		})
	}
}
