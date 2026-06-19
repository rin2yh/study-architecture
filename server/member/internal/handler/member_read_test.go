package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func TestListMembers(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.members (email, display_name) VALUES ($1, $2)`,
		"user@example.com", "サンプル会員"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newServer(rdb.NewMemberQuery(pool), rdb.NewMemberCommand(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Member
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []api.Member{{Email: "user@example.com", DisplayName: "サンプル会員"}}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Member{}, "Id", "CreatedAt")); diff != "" {
		t.Fatalf("members mismatch (-want +got):\n%s", diff)
	}
}

func TestListMembersError(t *testing.T) {
	repo := stub.Repo{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo, repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "internal" {
		t.Fatalf("code = %q, want internal", body.Code)
	}
	if body.Message == "db failure" {
		t.Fatalf("message must not expose internal error: %q", body.Message)
	}
}

func TestGetMember(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	member := db.MemberMember{ID: 1, Email: "user@example.com", DisplayName: "サンプル会員", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
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
		{"正常系 会員を返す", args{stub.Repo{Member: member}, "/members/1"}, want{http.StatusOK, ""}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/members/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/members/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newServer(tt.args.repo, tt.args.repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Member
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Email != "user@example.com" {
				t.Fatalf("unexpected member: %+v", got)
			}
		})
	}
}
