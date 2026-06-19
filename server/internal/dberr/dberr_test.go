package dberr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
)

func TestFromRead(t *testing.T) {
	other := errors.New("boom")
	tests := []struct {
		name string
		in   error
		want error // errors.Is で照合。nil はエラー無し
	}{
		{"正常系 エラー無しは nil", nil, nil},
		{"準正常系 no rows は ErrNotFound に正規化", pgx.ErrNoRows, dberr.ErrNotFound},
		{"準正常系 ラップされた no rows も ErrNotFound", fmt.Errorf("get: %w", pgx.ErrNoRows), dberr.ErrNotFound},
		{"異常系 その他エラーは透過", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dberr.FromRead(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("FromRead = %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("FromRead = %v, want errors.Is %v", got, tt.want)
			}
		})
	}
}

func TestFromWrite(t *testing.T) {
	other := errors.New("boom")
	uniqueViolation := &pgconn.PgError{Code: "23505"}
	fkViolation := &pgconn.PgError{Code: "23503"}
	tests := []struct {
		name string
		in   error
		want error // errors.Is で照合。nil はエラー無し
	}{
		{"正常系 エラー無しは nil", nil, nil},
		{"準正常系 unique_violation は ErrConflict に正規化", uniqueViolation, dberr.ErrConflict},
		{"準正常系 ラップされた unique_violation も ErrConflict", fmt.Errorf("insert: %w", uniqueViolation), dberr.ErrConflict},
		{"異常系 別の SQLSTATE は透過", fkViolation, fkViolation},
		{"異常系 その他エラーは透過", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dberr.FromWrite(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("FromWrite = %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("FromWrite = %v, want errors.Is %v", got, tt.want)
			}
		})
	}
}

func TestFromUpdate(t *testing.T) {
	other := errors.New("boom")
	uniqueViolation := &pgconn.PgError{Code: "23505"}
	fkViolation := &pgconn.PgError{Code: "23503"}
	tests := []struct {
		name string
		in   error
		want error // errors.Is で照合。nil はエラー無し
	}{
		{"正常系 エラー無しは nil", nil, nil},
		{"準正常系 no rows は ErrNotFound に正規化", pgx.ErrNoRows, dberr.ErrNotFound},
		{"準正常系 ラップされた no rows も ErrNotFound", fmt.Errorf("update: %w", pgx.ErrNoRows), dberr.ErrNotFound},
		{"準正常系 unique_violation は ErrConflict に正規化", uniqueViolation, dberr.ErrConflict},
		{"異常系 別の SQLSTATE は透過", fkViolation, fkViolation},
		{"異常系 その他エラーは透過", other, other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dberr.FromUpdate(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("FromUpdate = %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("FromUpdate = %v, want errors.Is %v", got, tt.want)
			}
		})
	}
}
