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
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func TestListOrders(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
		int64(10), "paid", int64(5000)); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newServer(repository.NewRepository(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []api.Order{{MemberId: 10, Status: "paid", TotalCents: 5000}}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Order{}, "Id", "CreatedAt")); diff != "" {
		t.Fatalf("orders mismatch (-want +got):\n%s", diff)
	}
}

func TestListOrdersFilter(t *testing.T) {
	all := []db.OrderOrder{
		{ID: 1, MemberID: 10, Status: "paid", TotalCents: 5000},
		{ID: 2, MemberID: 20, Status: "paid", TotalCents: 3000},
	}
	mine := []db.OrderOrder{{ID: 1, MemberID: 10, Status: "paid", TotalCents: 5000}}
	type args struct {
		repo   stub.Repo
		header string
	}
	type want struct {
		count       int
		firstMember int64
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 X-Member-Id 付きは本人分だけ返す", args{stub.Repo{Orders: all, ByMember: mine}, "10"}, want{1, 10}},
		{"準正常系 ヘッダ無しは全件返す", args{stub.Repo{Orders: all, ByMember: mine}, ""}, want{2, 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			if tt.args.header != "" {
				req.Header.Set("X-Member-Id", tt.args.header)
			}
			rec := httptest.NewRecorder()
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
			}
			var got []api.Order
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(got) != tt.want.count {
				t.Fatalf("count = %d, want %d", len(got), tt.want.count)
			}
			if got[0].MemberId != tt.want.firstMember {
				t.Fatalf("first memberId = %d, want %d", got[0].MemberId, tt.want.firstMember)
			}
		})
	}
}

func TestListOrdersError(t *testing.T) {
	repo := stub.Repo{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

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

func TestGetOrder(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	order := db.OrderOrder{ID: 1, MemberID: 10, Status: "paid", TotalCents: 5000, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
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
		{"正常系 注文を返す", args{stub.Repo{Order: order}, "/orders/1"}, want{http.StatusOK, ""}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/orders/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/orders/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newServer(tt.args.repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Order
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.MemberId != 10 {
				t.Fatalf("unexpected order: %+v", got)
			}
		})
	}
}
