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
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/auth"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
	"github.com/rin2yh/study-architecture/server/member/internal/handler"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func memberWithPassword(t *testing.T, plain string) db.MemberMember {
	t.Helper()
	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return db.MemberMember{ID: 7, Email: "user@example.com", DisplayName: "会員", PasswordHash: hash}
}

func TestCreateSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.members (email, display_name, password_hash) VALUES ($1, $2, $3)`,
		"user@example.com", "会員", hash); err != nil {
		t.Fatalf("insert member: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader([]byte(`{"email":"user@example.com","password":"password123"}`)))
	req.Header.Set("Content-Type", "application/json")
	newServer(handler.New(rdb.NewMemberQuery(pool), rdb.NewMemberCommand(pool))).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Id == "" {
		t.Fatal("session id (生トークン) が空")
	}
	if got.MemberId != 1 {
		t.Fatalf("memberId = %d, want 1", got.MemberId)
	}
}

func TestCreateSessionError(t *testing.T) {
	withPassword := memberWithPassword(t, "password123")
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
		{"異常系 password 不一致は 401 unauthorized", args{stub.MemberStub{Member: withPassword}, `{"email":"user@example.com","password":"wrong-password"}`}, want{http.StatusUnauthorized, "unauthorized"}},
		{"異常系 未登録 email は 401 unauthorized", args{stub.MemberStub{Err: dberr.ErrNotFound}, `{"email":"none@example.com","password":"password123"}`}, want{http.StatusUnauthorized, "unauthorized"}},
		{"異常系 email 欠落は 400 bad_request", args{stub.MemberStub{}, `{"password":"password123"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 会員照会の DB エラーは 500 internal", args{stub.MemberStub{Err: errors.New("db failure")}, `{"email":"user@example.com","password":"password123"}`}, want{http.StatusInternalServerError, "internal"}},
		{"異常系 セッション発行の DB エラーは 500 internal", args{stub.MemberStub{Member: withPassword, SessionErr: errors.New("db failure")}, `{"email":"user@example.com","password":"password123"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(handler.New(tt.args.fake, tt.args.fake)).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestDeleteSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	var memberID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO member.members (email, display_name, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		"user@example.com", "会員", "stored-hash").Scan(&memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.sessions (id, member_id, expires_at) VALUES ($1, $2, now() + interval '1 hour')`,
		auth.HashToken("raw-token"), memberID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	server := newWriteServer(rdb.NewMemberCommand(pool))

	t.Run("正常系 破棄して 204", func(t *testing.T) {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/raw-token", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body.String())
		}
	})
	t.Run("準正常系 未存在でも冪等に 204", func(t *testing.T) {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/raw-token", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

func TestDeleteSessionError(t *testing.T) {
	rec := httptest.NewRecorder()
	fake := stub.MemberStub{SessionErr: errors.New("db failure")}
	newWriteServer(fake).ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/sessions/raw-token", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body: %s)", rec.Code, rec.Body.String())
	}
	apitest.AssertErrorCode(t, rec.Body.Bytes(), "internal")
}
