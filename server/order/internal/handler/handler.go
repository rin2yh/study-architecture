package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
)

type Query interface {
	ListOrders(ctx context.Context) ([]db.OrderOrder, error)
	ListOrdersByMember(ctx context.Context, memberID int64) ([]db.OrderOrder, error)
	GetOrder(ctx context.Context, id int64) (db.OrderOrder, error)
	GetOrderItems(ctx context.Context, orderID int64) ([]db.OrderOrderItem, error)
}

type Command interface {
	CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.OrderOrder, error)
	UpdateOrder(ctx context.Context, arg db.UpdateOrderParams) (db.OrderOrder, error)
	Checkout(ctx context.Context, memberID int64, status string, totalCents int64, lines []rdb.CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error)
	DeleteOrder(ctx context.Context, id int64) error
	CancelOrder(ctx context.Context, id int64, traceparent string) (db.OrderOrder, error)
}

type readHandler struct {
	query Query
}

type writeHandler struct {
	command   Command
	product   gateway.ProductPort
	payment   gateway.PaymentPort
	inventory gateway.InventoryPort
}

type Handler struct {
	*readHandler
	*writeHandler
}

var _ api.ServerInterface = (*Handler)(nil)

func New(query Query, command Command, product gateway.ProductPort, payment gateway.PaymentPort, inventory gateway.InventoryPort) *Handler {
	return &Handler{
		readHandler:  &readHandler{query: query},
		writeHandler: &writeHandler{command: command, product: product, payment: payment, inventory: inventory},
	}
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
