package gateway_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
)

func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestOrderClientFetchDestination(t *testing.T) {
	const withAddr = `{"id":20,"memberId":1,"status":"confirmed","totalCents":2500,"createdAt":"2026-01-01T00:00:00Z","shippingAddress":{"recipient":"山田太郎","postalCode":"1500001","prefecture":"東京都","city":"渋谷区","line1":"神宮前1-2-3"}}`
	const noAddr = `{"id":20,"memberId":1,"status":"confirmed","totalCents":2500,"createdAt":"2026-01-01T00:00:00Z"}`
	type want struct {
		dest    gateway.Destination
		errIs   error
		wantErr bool
	}
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    want
	}{
		{
			"正常系 200 の shippingAddress を Destination に複写",
			jsonHandler(http.StatusOK, withAddr),
			want{dest: gateway.Destination{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}},
		},
		{
			"準正常系 宛先未設定の注文は空 Destination (error なし)",
			jsonHandler(http.StatusOK, noAddr),
			want{dest: gateway.Destination{}},
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
			t.Setenv("ORDER_API_URL", srv.URL)
			c, err := gateway.NewOrderClient()
			if err != nil {
				t.Fatalf("NewOrderClient: %v", err)
			}

			got, err := c.FetchDestination(t.Context(), mustOrderID(t, "20"))
			if tt.want.wantErr {
				if !errors.Is(err, tt.want.errIs) {
					t.Fatalf("err = %v, want errors.Is %v", err, tt.want.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchDestination: %v", err)
			}
			if got != tt.want.dest {
				t.Fatalf("dest = %+v, want %+v", got, tt.want.dest)
			}
		})
	}
}

func TestOrderClientFetchDestinationTransportError(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, `{}`))
	t.Setenv("ORDER_API_URL", srv.URL)
	c, err := gateway.NewOrderClient()
	if err != nil {
		t.Fatalf("NewOrderClient: %v", err)
	}
	srv.Close()

	if _, err := c.FetchDestination(t.Context(), mustOrderID(t, "20")); !errors.Is(err, gateway.ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
}

func mustOrderID(t *testing.T, raw string) order.ID {
	t.Helper()
	id, err := order.Parse(raw)
	if err != nil {
		t.Fatalf("order.Parse(%q): %v", raw, err)
	}
	return id
}
