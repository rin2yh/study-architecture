package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

type Handler struct {
	repo     repository.OrderRepository
	product  gateway.ProductPort
	payment  gateway.PaymentPort
	shipping gateway.ShippingPort
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.OrderRepository, product gateway.ProductPort, payment gateway.PaymentPort, shipping gateway.ShippingPort) *Handler {
	return &Handler{repo: repo, product: product, payment: payment, shipping: shipping}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

func toAPIOrderWithItems(r db.OrderOrder, items []db.OrderOrderItem) api.Order {
	o := toAPIOrder(r)
	out := make([]api.OrderItem, 0, len(items))
	for _, it := range items {
		out = append(out, api.OrderItem{
			ProductId:      it.ProductID,
			ProductName:    it.ProductName,
			UnitPriceCents: it.UnitPriceCents,
			Quantity:       int(it.Quantity),
		})
	}
	o.Items = &out
	return o
}
