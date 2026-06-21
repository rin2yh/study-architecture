package assert

import (
	"encoding/json"
	"testing"
)

func ErrorCode(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if e.Code != wantCode {
		t.Fatalf("code = %q, want %q", e.Code, wantCode)
	}
}
