// Package dberr は 5 サービス共通で、postgres ドライバ (pgx) 固有のエラーを
// rdb 層が返すセンチネルエラーに正規化する。handler はこのセンチネルを見て
// middleware.NotFound / Conflict に対応づけるため、db 依存が handler に漏れない。
package dberr

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrNotFound は対象行が存在しない (読み取りで no rows)。handler は 404 に対応づける。
	ErrNotFound = errors.New("not found")
	// ErrConflict は一意制約違反など状態の競合。handler は 409 に対応づける。
	ErrConflict = errors.New("conflict")
)

// FromRead は読み取りクエリのエラーを正規化する。no rows は ErrNotFound に、
// それ以外はそのまま返す。
func FromRead(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// FromWrite は書き込みクエリのエラーを正規化する。unique_violation (SQLSTATE 23505)
// は ErrConflict に、それ以外はそのまま返す。
func FromWrite(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}

// 更新は対象行が無ければ no rows、unique 列を既存値に変えれば 23505 と、読み取り・書き込み
// 双方の失敗が起こりうる。FromRead / FromWrite の片方では拾えないため両者を兼ねる。
func FromUpdate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return FromWrite(err)
}

// FromInsertSkipped は INSERT ... ON CONFLICT DO NOTHING の :one が行を返さない (既存と衝突して
// 挿入がスキップされた) を ErrConflict に正規化する。read の no rows (ErrNotFound) とは逆の意味。
func FromInsertSkipped(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	return err
}
