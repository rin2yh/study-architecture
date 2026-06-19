package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/auth"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

// UI 側の Cookie Max-Age (client/app/mypage/src/session.ts) と揃える必要がある。
// ずれると Cookie だけ先に失効する/逆に死んだセッションを送り続けることになる。
const sessionTTL = 7 * 24 * time.Hour

// invalidCredentials は「メール無し」と「パスワード不一致」で文言を変えない。
// 差異から会員の存在有無を推測される (user enumeration) のを防ぐため。
const invalidCredentials = "invalid email or password"

func (h *Handler) CreateSession(c *gin.Context) {
	var req api.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	member, err := h.repo.GetMemberByEmail(c.Request.Context(), string(req.Email))
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.Unauthorized(invalidCredentials))
			return
		}
		_ = c.Error(err)
		return
	}
	if err := auth.VerifyPassword(member.PasswordHash, req.Password); err != nil {
		_ = c.Error(middleware.Unauthorized(invalidCredentials))
		return
	}

	token, id, err := auth.NewSessionToken()
	if err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.repo.CreateSession(c.Request.Context(), db.CreateSessionParams{
		ID:        id,
		MemberID:  member.ID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(sessionTTL), Valid: true},
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPISession(token, row))
}

func (h *Handler) DeleteSession(c *gin.Context, id api.SessionIdPath) {
	if err := h.repo.DeleteSession(c.Request.Context(), auth.HashToken(id)); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
