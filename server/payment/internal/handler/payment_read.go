package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/payment/api"
)

func (h *Handler) ListPayments(c *gin.Context) {
	rows, err := h.query.ListPayments(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Payment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPIPayment(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetPayment(c *gin.Context, id api.IdPath) {
	row, err := h.query.GetPayment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("payment not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIPayment(row))
}
