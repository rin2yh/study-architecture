package handler_test

import (
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
	"github.com/rin2yh/study-architecture/server/member/internal/auth"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func TestGetSession(t *testing.T) {
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

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewMemberQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions/raw-token", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cmptest.Equal(t, api.Session{Id: "raw-token", MemberId: memberID}, got, "ExpiresAt")
}

func TestGetSessionError(t *testing.T) {
	type args struct {
		fake stub.MemberStub
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
		{"異常系 未存在/期限切れは 404 not_found", args{stub.MemberStub{SessionErr: dberr.ErrNotFound}}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.MemberStub{SessionErr: errors.New("db failure")}}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newReadServer(tt.args.fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sessions/raw-token", nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
