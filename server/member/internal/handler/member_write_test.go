package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	"github.com/rin2yh/study-architecture/server/internal/test/cmptest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/handler"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command))
}

func TestCreateMember(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	if _, err := pool.Exec(t.Context(), `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/members", bytes.NewReader([]byte(`{"email":"new@example.com","displayName":"新規会員","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewMemberCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Member
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cmptest.Equal(t, api.Member{Email: "new@example.com", DisplayName: "新規会員"}, got, "Id", "CreatedAt")
}

func TestCreateMemberError(t *testing.T) {
	type args struct {
		fake stub.MemberStub
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
		{"異常系 displayName 欠落は 400 bad_request", args{stub.MemberStub{}, `{"email":"new@example.com","password":"password123"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 email 形式不正は 400 bad_request", args{stub.MemberStub{}, `{"email":"not-an-email","displayName":"x","password":"password123"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 password が短いと 400 bad_request", args{stub.MemberStub{}, `{"email":"new@example.com","displayName":"x","password":"short"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 email 重複は 409 conflict", args{stub.MemberStub{Err: dberr.ErrConflict}, `{"email":"dup@example.com","displayName":"重複","password":"password123"}`}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.MemberStub{Err: errors.New("db failure")}, `{"email":"x@example.com","displayName":"x","password":"password123"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/members", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestUpdateMember(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.members (email, display_name) VALUES ($1, $2)`,
		"user@example.com", "サンプル会員"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/members/1", bytes.NewReader([]byte(`{"email":"upd@example.com","displayName":"更新後会員"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewMemberCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Member
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cmptest.Equal(t, api.Member{Email: "upd@example.com", DisplayName: "更新後会員"}, got, "Id", "CreatedAt")
}

func TestUpdateMemberError(t *testing.T) {
	type args struct {
		fake stub.MemberStub
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
		{"異常系 displayName 欠落は 400 bad_request", args{stub.MemberStub{}, "/members/1", `{"email":"upd@example.com"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 email 形式不正は 400 bad_request", args{stub.MemberStub{}, "/members/1", `{"email":"not-an-email","displayName":"x"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.MemberStub{Err: dberr.ErrNotFound}, "/members/99", `{"email":"upd@example.com","displayName":"x"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 email 重複は 409 conflict", args{stub.MemberStub{Err: dberr.ErrConflict}, "/members/1", `{"email":"dup@example.com","displayName":"重複"}`}, want{http.StatusConflict, "conflict"}},
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
