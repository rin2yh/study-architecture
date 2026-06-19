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
	"github.com/rin2yh/study-architecture/server/member/internal/db"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func TestCreateMember(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	created := db.MemberMember{ID: 10, Email: "new@example.com", DisplayName: "新規会員", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
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
		{"正常系 会員を作成し 201", args{stub.Repo{Member: created}, `{"email":"new@example.com","displayName":"新規会員"}`}, want{http.StatusCreated, ""}},
		{"異常系 displayName 欠落は 400 bad_request", args{stub.Repo{}, `{"email":"new@example.com"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 email 形式不正は 400 bad_request", args{stub.Repo{}, `{"email":"not-an-email","displayName":"x"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 email 重複は 409 conflict", args{stub.Repo{Err: dberr.ErrConflict}, `{"email":"dup@example.com","displayName":"重複"}`}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, `{"email":"x@example.com","displayName":"x"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/members", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Member
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 10 || got.Email != "new@example.com" {
				t.Fatalf("unexpected member: %+v", got)
			}
		})
	}
}

func TestUpdateMember(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := db.MemberMember{ID: 1, Email: "upd@example.com", DisplayName: "更新後会員", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		repo stub.Repo
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
		{"正常系 会員を更新し 200", args{stub.Repo{Member: updated}, "/members/1", `{"email":"upd@example.com","displayName":"更新後会員"}`}, want{http.StatusOK, ""}},
		{"異常系 displayName 欠落は 400 bad_request", args{stub.Repo{}, "/members/1", `{"email":"upd@example.com"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 email 形式不正は 400 bad_request", args{stub.Repo{}, "/members/1", `{"email":"not-an-email","displayName":"x"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/members/99", `{"email":"upd@example.com","displayName":"x"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 email 重複は 409 conflict", args{stub.Repo{Err: dberr.ErrConflict}, "/members/1", `{"email":"dup@example.com","displayName":"重複"}`}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/members/1", `{"email":"x@example.com","displayName":"x"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, tt.args.path, bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Member
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Email != "upd@example.com" {
				t.Fatalf("unexpected member: %+v", got)
			}
		})
	}
}
