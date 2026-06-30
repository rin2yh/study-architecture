package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
	"github.com/rin2yh/study-architecture/server/member/internal/stub"
)

func TestListAddresses(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_MEMBER")
	ctx := t.Context()
	memberID := seedMember(t, pool)
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.addresses (member_id, recipient, postal_code, prefecture, city, line1)
		 VALUES ($1, '山田太郎', '1500001', '東京都', '渋谷区', '神宮前1-2-3')`, memberID); err != nil {
		t.Fatalf("insert address: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewMemberQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members/1/addresses", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got []api.Address
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []api.Address{{Id: 1, MemberId: 1, Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}}
	assert.DeepEqualSlice(t, want, got, "CreatedAt")
}

func TestListAddressesError(t *testing.T) {
	rec := httptest.NewRecorder()
	newReadServer(stub.MemberStub{AddressErr: errors.New("db failure")}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members/1/addresses", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	assert.ErrorCode(t, rec.Body.Bytes(), "internal")
}

func TestGetAddress(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_MEMBER")
	ctx := t.Context()
	memberID := seedMember(t, pool)
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.addresses (member_id, recipient, postal_code, prefecture, city, line1)
		 VALUES ($1, '山田太郎', '1500001', '東京都', '渋谷区', '神宮前1-2-3')`, memberID); err != nil {
		t.Fatalf("insert address: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewMemberQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members/1/addresses/1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Address
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Address{Id: 1, MemberId: 1, Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}
	assert.DeepEqual(t, want, got, "CreatedAt")
}

func TestGetAddressError(t *testing.T) {
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
		{"準正常系 未存在は 404 not_found", args{stub.MemberStub{AddressErr: dberr.ErrNotFound}}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.MemberStub{AddressErr: errors.New("db failure")}}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newReadServer(tt.args.fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/members/1/addresses/1", nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
