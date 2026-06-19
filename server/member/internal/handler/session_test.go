package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/auth"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
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
	expires := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	created := memberWithPassword(t, "password123")
	session := db.MemberSession{MemberID: 7, ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true}}

	type args struct {
		repo stub.Repo
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
		{"正常系 認証成功で 201", args{stub.Repo{Member: created, Session: session}, `{"email":"user@example.com","password":"password123"}`}, want{http.StatusCreated, ""}},
		{"異常系 password 不一致は 401 unauthorized", args{stub.Repo{Member: created}, `{"email":"user@example.com","password":"wrong-password"}`}, want{http.StatusUnauthorized, "unauthorized"}},
		{"異常系 未登録 email は 401 unauthorized", args{stub.Repo{Err: dberr.ErrNotFound}, `{"email":"none@example.com","password":"password123"}`}, want{http.StatusUnauthorized, "unauthorized"}},
		{"異常系 email 欠落は 400 bad_request", args{stub.Repo{}, `{"password":"password123"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 GetMemberByEmail の DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, `{"email":"user@example.com","password":"password123"}`}, want{http.StatusInternalServerError, "internal"}},
		{"異常系 CreateSession の DB エラーは 500 internal", args{stub.Repo{Member: created, SessionErr: errors.New("db failure")}, `{"email":"user@example.com","password":"password123"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Session
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id == "" {
				t.Fatal("session id (生トークン) が空")
			}
			if got.MemberId != 7 {
				t.Fatalf("memberId = %d, want 7", got.MemberId)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	expires := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	session := db.MemberSession{MemberID: 7, ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true}}

	type args struct {
		repo stub.Repo
		path string
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
		{"正常系 有効なセッションは 200", args{stub.Repo{Session: session}, "/sessions/raw-token"}, want{http.StatusOK, ""}},
		{"異常系 未存在/期限切れは 404 not_found", args{stub.Repo{SessionErr: dberr.ErrNotFound}, "/sessions/raw-token"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{SessionErr: errors.New("db failure")}, "/sessions/raw-token"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.args.path, nil)
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Session
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			// 渡した生トークンがそのまま id として返る (DB はハッシュしか持たない)。
			if got.Id != "raw-token" || got.MemberId != 7 {
				t.Fatalf("unexpected session: %+v", got)
			}
		})
	}
}

func TestDeleteSession(t *testing.T) {
	type args struct {
		repo stub.Repo
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
		{"正常系 破棄して 204", args{stub.Repo{}}, want{http.StatusNoContent, ""}},
		{"準正常系 未存在でも冪等に 204", args{stub.Repo{}}, want{http.StatusNoContent, ""}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{SessionErr: errors.New("db failure")}}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/sessions/raw-token", nil)
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
			}
		})
	}
}
