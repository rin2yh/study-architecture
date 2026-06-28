package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	"github.com/rin2yh/study-architecture/server/inventory/api"
	"github.com/rin2yh/study-architecture/server/inventory/internal/handler"
	"github.com/rin2yh/study-architecture/server/inventory/internal/stub"
)

func newReadServer(query handler.Query) http.Handler {
	return newServer(handler.New(query, nil))
}

func getAvailability(query handler.Query, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newReadServer(query).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestGetAvailability(t *testing.T) {
	t.Run("正常系 利用可能在庫を返す", func(t *testing.T) {
		rec := getAvailability(stub.InventoryStub{AvailableQty: 7}, "/availability/100")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
		var got api.Availability
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		assert.DeepEqual(t, api.Availability{ProductId: 100, Available: 7}, got)
	})

	t.Run("異常系 DB エラーは 500 internal", func(t *testing.T) {
		rec := getAvailability(stub.InventoryStub{Err: errors.New("db failure")}, "/availability/100")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500 (body: %s)", rec.Code, rec.Body.String())
		}
		assert.ErrorCode(t, rec.Body.Bytes(), "internal")
	})
}
