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

func (h *readHandler) ListAddresses(c *gin.Context, id api.IdPath) {
	rows, err := h.query.ListAddresses(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Address, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPIAddress(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *readHandler) GetAddress(c *gin.Context, id api.IdPath, addressID api.AddressIdPath) {
	row, err := h.query.GetAddress(c.Request.Context(), db.GetAddressParams{ID: addressID, MemberID: id})
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
