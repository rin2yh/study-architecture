package rdb

import (
	"time"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

// port を sqlc 生成型から切り離す (ADR-[[202607011200]])。pgtype を標準型へ寄せ、schema 変更を
// rdb のマッピングに閉じ込める。PasswordHash はログイン照合で command 層外 (handler) が読むため載せる。
type Member struct {
	ID           int64
	Email        string
	DisplayName  string
	PasswordHash string
	CreatedAt    time.Time
}

type Session struct {
	ID        string
	MemberID  int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Address struct {
	ID         int64
	MemberID   int64
	Recipient  string
	PostalCode string
	Prefecture string
	City       string
	Line1      string
	CreatedAt  time.Time
}

type MemberCreate struct {
	Email        string
	DisplayName  string
	PasswordHash string
}

type MemberUpdate struct {
	ID          int64
	Email       string
	DisplayName string
}

type SessionCreate struct {
	ID        string
	MemberID  int64
	ExpiresAt time.Time
}

type AddressCreate struct {
	MemberID   int64
	Recipient  string
	PostalCode string
	Prefecture string
	City       string
	Line1      string
}

type AddressUpdate struct {
	ID         int64
	MemberID   int64
	Recipient  string
	PostalCode string
	Prefecture string
	City       string
	Line1      string
}

type AddressRef struct {
	ID       int64
	MemberID int64
}

func toMember(r db.MemberMember) Member {
	return Member{
		ID:           r.ID,
		Email:        r.Email,
		DisplayName:  r.DisplayName,
		PasswordHash: r.PasswordHash,
		CreatedAt:    r.CreatedAt.Time,
	}
}

func toMembers(rows []db.MemberMember) []Member {
	out := make([]Member, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMember(r))
	}
	return out
}

func toSession(r db.MemberSession) Session {
	return Session{
		ID:        r.ID,
		MemberID:  r.MemberID,
		ExpiresAt: r.ExpiresAt.Time,
		CreatedAt: r.CreatedAt.Time,
	}
}

func toAddress(r db.MemberAddress) Address {
	return Address{
		ID:         r.ID,
		MemberID:   r.MemberID,
		Recipient:  r.Recipient,
		PostalCode: r.PostalCode,
		Prefecture: r.Prefecture,
		City:       r.City,
		Line1:      r.Line1,
		CreatedAt:  r.CreatedAt.Time,
	}
}

func toAddresses(rows []db.MemberAddress) []Address {
	out := make([]Address, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAddress(r))
	}
	return out
}
