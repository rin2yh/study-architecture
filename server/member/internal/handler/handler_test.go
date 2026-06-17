package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-service-base-architecture/server/member/api"
	"github.com/rin2yh/study-service-base-architecture/server/member/internal/db"
)

type stubRepo struct {
	members []db.MemberMember
	err     error
}

func (s stubRepo) ListMembers(context.Context) ([]db.MemberMember, error) {
	return s.members, s.err
}

func newServer(repo stubRepo) http.Handler {
	si := api.NewStrictHandler(New(repo), nil)
	return api.HandlerFromMux(si, http.NewServeMux())
}

func TestGetHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(stubRepo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body api.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, want ok", body.Status)
	}
}

func TestListMembers(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := stubRepo{members: []db.MemberMember{
		{ID: 1, Email: "user@example.com", DisplayName: "サンプル会員", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Member
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Email != "user@example.com" || got[0].DisplayName != "サンプル会員" || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected member: %+v", got[0])
	}
}

func TestListMembersError(t *testing.T) {
	repo := stubRepo{err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
