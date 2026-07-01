package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/auth"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
)

// ADR-[[202606211100]]
const sessionTTL = 7 * 24 * time.Hour

// ADR-[[202606211100]]
const invalidCredentials = "invalid email or password"

func (h *writeHandler) CreateSession(c *gin.Context) {
	var req api.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	member, err := h.query.GetMemberByEmail(c.Request.Context(), string(req.Email))
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
	row, err := h.command.CreateSession(c.Request.Context(), rdb.SessionCreate{
		ID:        id,
		MemberID:  member.ID,
		ExpiresAt: time.Now().Add(sessionTTL),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPISession(token, row))
}

func (h *writeHandler) DeleteSession(c *gin.Context, id api.SessionIdPath) {
	if err := h.command.DeleteSession(c.Request.Context(), auth.HashToken(id)); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
