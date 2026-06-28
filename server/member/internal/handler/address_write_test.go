package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func seedMember(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO member.members (email, display_name) VALUES ($1, $2) RETURNING id`,
		"user@example.com", "サンプル会員").Scan(&id); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	return id
}

const validAddress = `{"recipient":"山田太郎","postalCode":"1500001","prefecture":"東京都","city":"渋谷区","line1":"神宮前1-2-3"}`

func TestCreateAddress(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_MEMBER")
	seedMember(t, pool)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/members/1/addresses", bytes.NewReader([]byte(validAddress)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewMemberCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Address
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Address{MemberId: 1, Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}
	assert.DeepEqual(t, want, got, "Id", "CreatedAt")
}

func TestCreateAddressError(t *testing.T) {
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
		{"準正常系 recipient 欠落は 400 bad_request", args{stub.MemberStub{}, `{"postalCode":"1500001","prefecture":"東京都","city":"渋谷区","line1":"神宮前1-2-3"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 line1 欠落は 400 bad_request", args{stub.MemberStub{}, `{"recipient":"山田太郎","postalCode":"1500001","prefecture":"東京都","city":"渋谷区"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 DB エラーは 500 internal", args{stub.MemberStub{AddressErr: errors.New("db failure")}, validAddress}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/members/1/addresses", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestUpdateAddress(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_MEMBER")
	ctx := t.Context()
	memberID := seedMember(t, pool)
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.addresses (member_id, recipient, postal_code, prefecture, city, line1)
		 VALUES ($1, '旧名', '1500001', '東京都', '渋谷区', '旧1-2-3')`, memberID); err != nil {
		t.Fatalf("insert address: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/members/1/addresses/1", bytes.NewReader([]byte(`{"recipient":"新名","postalCode":"1000001","prefecture":"東京都","city":"千代田区","line1":"丸の内1-1-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewMemberCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Address
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Address{Id: 1, MemberId: 1, Recipient: "新名", PostalCode: "1000001", Prefecture: "東京都", City: "千代田区", Line1: "丸の内1-1-1"}
	assert.DeepEqual(t, want, got, "CreatedAt")
}

func TestUpdateAddressError(t *testing.T) {
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
		{"準正常系 city 欠落は 400 bad_request", args{stub.MemberStub{}, `{"recipient":"x","postalCode":"x","prefecture":"x","line1":"x"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 未存在は 404 not_found", args{stub.MemberStub{AddressErr: dberr.ErrNotFound}, validAddress}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.MemberStub{AddressErr: errors.New("db failure")}, validAddress}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/members/1/addresses/1", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestDeleteAddress(t *testing.T) {
	type args struct {
		fake stub.MemberStub
	}
	type want struct {
		status int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 削除は 204", args{stub.MemberStub{}}, want{http.StatusNoContent}},
		{"準正常系 未存在の削除も冪等に 204", args{stub.MemberStub{}}, want{http.StatusNoContent}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newWriteServer(tt.args.fake).ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/members/1/addresses/1", nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
		})
	}
}

func TestDeleteAddressError(t *testing.T) {
	rec := httptest.NewRecorder()
	newWriteServer(stub.MemberStub{AddressErr: errors.New("db failure")}).ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/members/1/addresses/1", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	assert.ErrorCode(t, rec.Body.Bytes(), "internal")
}
