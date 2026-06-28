package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type Query interface {
	ListMembers(ctx context.Context) ([]db.MemberMember, error)
	GetMember(ctx context.Context, id int64) (db.MemberMember, error)
	GetMemberByEmail(ctx context.Context, email string) (db.MemberMember, error)
	GetSession(ctx context.Context, id string) (db.MemberSession, error)
	ListAddresses(ctx context.Context, memberID int64) ([]db.MemberAddress, error)
	GetAddress(ctx context.Context, arg db.GetAddressParams) (db.MemberAddress, error)
}

type Command interface {
	CreateMember(ctx context.Context, arg db.CreateMemberParams) (db.MemberMember, error)
	UpdateMember(ctx context.Context, arg db.UpdateMemberParams) (db.MemberMember, error)
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.MemberSession, error)
	DeleteSession(ctx context.Context, id string) error
	CreateAddress(ctx context.Context, arg db.CreateAddressParams) (db.MemberAddress, error)
	UpdateAddress(ctx context.Context, arg db.UpdateAddressParams) (db.MemberAddress, error)
	DeleteAddress(ctx context.Context, arg db.DeleteAddressParams) error
}

type readHandler struct {
	query Query
}

// ログイン (CreateSession) は会員照合 (query) とセッション発行 (command) の双方を要するため
// writeHandler は query も持つ。
type writeHandler struct {
	query   Query
	command Command
}

type Handler struct {
	*readHandler
	*writeHandler
}

var _ api.ServerInterface = (*Handler)(nil)

func New(query Query, command Command) *Handler {
	return &Handler{
		readHandler:  &readHandler{query: query},
		writeHandler: &writeHandler{query: query, command: command},
	}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func toAPIMember(r db.MemberMember) api.Member {
	return api.Member{
		Id:          r.ID,
		Email:       r.Email,
		DisplayName: r.DisplayName,
		CreatedAt:   r.CreatedAt.Time,
	}
}

func toAPIAddress(r db.MemberAddress) api.Address {
	return api.Address{
		Id:         r.ID,
		MemberId:   r.MemberID,
		Recipient:  r.Recipient,
		PostalCode: r.PostalCode,
		Prefecture: r.Prefecture,
		City:       r.City,
		Line1:      r.Line1,
		CreatedAt:  r.CreatedAt.Time,
	}
}

func toAPISession(token string, r db.MemberSession) api.Session {
	return api.Session{
		Id:        token,
		MemberId:  r.MemberID,
		ExpiresAt: r.ExpiresAt.Time,
	}
}
