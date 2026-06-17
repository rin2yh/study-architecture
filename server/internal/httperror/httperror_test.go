package httperror

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	return c, rec
}

func TestRequestErrorHandler(t *testing.T) {
	c, rec := newCtx()
	RequestErrorHandler(c, errors.New("missing required field"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "bad_request" {
		t.Fatalf("code = %q", body.Code)
	}
	if body.Message != "missing required field" {
		t.Fatalf("message = %q", body.Message)
	}
}

func TestHandlerErrorHandler(t *testing.T) {
	c, rec := newCtx()
	HandlerErrorHandler(c, errors.New("connection refused"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "internal" {
		t.Fatalf("code = %q, want internal", body.Code)
	}
	if body.Message == "connection refused" {
		t.Fatalf("message must not expose internal: %q", body.Message)
	}
}

func TestResponseErrorHandler(t *testing.T) {
	c, rec := newCtx()
	ResponseErrorHandler(c, errors.New("encode fail"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "internal" {
		t.Fatalf("code = %q, want internal", body.Code)
	}
	if body.Message == "encode fail" {
		t.Fatalf("message must not expose internal: %q", body.Message)
	}
}
