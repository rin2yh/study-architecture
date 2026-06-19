package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

type Handler struct {
	repo repository.OrderRepository
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.OrderRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListOrders(c *gin.Context) {
	rows, err := h.repo.ListOrders(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Order, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPIOrder(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetOrder(c *gin.Context, id api.IdPath) {
	row, err := h.repo.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("order not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIOrder(row))
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req api.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.TotalCents < 0 {
		_ = c.Error(middleware.Unprocessable("totalCents must not be negative"))
		return
	}
	row, err := h.repo.CreateOrder(c.Request.Context(), db.CreateOrderParams{
		MemberID:   req.MemberId,
		Status:     req.Status,
		TotalCents: req.TotalCents,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIOrder(row))
}

func (h *Handler) UpdateOrder(c *gin.Context, id api.IdPath) {
	var req api.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.TotalCents < 0 {
		_ = c.Error(middleware.Unprocessable("totalCents must not be negative"))
		return
	}
	row, err := h.repo.UpdateOrder(c.Request.Context(), db.UpdateOrderParams{
		ID:         id,
		Status:     req.Status,
		TotalCents: req.TotalCents,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("order not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIOrder(row))
}

func toAPIOrder(r db.OrderOrder) api.Order {
	return api.Order{
		Id:         r.ID,
		MemberId:   r.MemberID,
		Status:     r.Status,
		TotalCents: r.TotalCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}
