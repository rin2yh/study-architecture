package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
)

func (h *writeHandler) CreateAddress(c *gin.Context, id api.IdPath) {
	var req api.AddressInput
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.command.CreateAddress(c.Request.Context(), rdb.AddressCreate{
		MemberID:   id,
		Recipient:  req.Recipient,
		PostalCode: req.PostalCode,
		Prefecture: req.Prefecture,
		City:       req.City,
		Line1:      req.Line1,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("member not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIAddress(row))
}

func (h *writeHandler) UpdateAddress(c *gin.Context, id api.IdPath, addressID api.AddressIdPath) {
	var req api.AddressInput
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.command.UpdateAddress(c.Request.Context(), rdb.AddressUpdate{
		ID:         addressID,
		MemberID:   id,
		Recipient:  req.Recipient,
		PostalCode: req.PostalCode,
		Prefecture: req.Prefecture,
		City:       req.City,
		Line1:      req.Line1,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("address not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIAddress(row))
}

func (h *writeHandler) DeleteAddress(c *gin.Context, id api.IdPath, addressID api.AddressIdPath) {
	if err := h.command.DeleteAddress(c.Request.Context(), rdb.AddressRef{ID: addressID, MemberID: id}); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
