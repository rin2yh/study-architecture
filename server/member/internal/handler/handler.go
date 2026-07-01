package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
)

type Query interface {
	ListMembers(ctx context.Context) ([]rdb.Member, error)
	GetMember(ctx context.Context, id int64) (rdb.Member, error)
	GetMemberByEmail(ctx context.Context, email string) (rdb.Member, error)
	GetSession(ctx context.Context, id string) (rdb.Session, error)
	ListAddresses(ctx context.Context, memberID int64) ([]rdb.Address, error)
	GetAddress(ctx context.Context, ref rdb.AddressRef) (rdb.Address, error)
}

type Command interface {
	CreateMember(ctx context.Context, arg rdb.MemberCreate) (rdb.Member, error)
	UpdateMember(ctx context.Context, arg rdb.MemberUpdate) (rdb.Member, error)
	CreateSession(ctx context.Context, arg rdb.SessionCreate) (rdb.Session, error)
	DeleteSession(ctx context.Context, id string) error
	CreateAddress(ctx context.Context, arg rdb.AddressCreate) (rdb.Address, error)
	UpdateAddress(ctx context.Context, arg rdb.AddressUpdate) (rdb.Address, error)
	DeleteAddress(ctx context.Context, ref rdb.AddressRef) error
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

func toAPIMember(r rdb.Member) api.Member {
	return api.Member{
		Id:          r.ID,
		Email:       r.Email,
		DisplayName: r.DisplayName,
		CreatedAt:   r.CreatedAt,
	}
}

func toAPIAddress(r rdb.Address) api.Address {
	return api.Address{
		Id:         r.ID,
		MemberId:   r.MemberID,
		Recipient:  r.Recipient,
		PostalCode: r.PostalCode,
		Prefecture: r.Prefecture,
		City:       r.City,
		Line1:      r.Line1,
		CreatedAt:  r.CreatedAt,
	}
}

func toAPISession(token string, r rdb.Session) api.Session {
	return api.Session{
		Id:        token,
		MemberId:  r.MemberID,
		ExpiresAt: r.ExpiresAt,
	}
}
