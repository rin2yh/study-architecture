package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

func (h *Handler) CreateMember(c *gin.Context) {
	var req api.CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.repo.CreateMember(c.Request.Context(), db.CreateMemberParams{
		Email:       string(req.Email),
		DisplayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrConflict) {
			_ = c.Error(middleware.Conflict("member with this email already exists"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIMember(row))
}

func (h *Handler) UpdateMember(c *gin.Context, id api.IdPath) {
	var req api.UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.repo.UpdateMember(c.Request.Context(), db.UpdateMemberParams{
		ID:          id,
		Email:       string(req.Email),
		DisplayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("member not found"))
			return
		}
		if errors.Is(err, dberr.ErrConflict) {
			_ = c.Error(middleware.Conflict("member with this email already exists"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIMember(row))
}
