// Package gateway は order が他サービス (product / payment / inventory) を呼ぶ出力ポートと
// 生成クライアント実装をまとめる。
package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/order/internal/client/inventory"
	"github.com/rin2yh/study-architecture/server/order/internal/client/payment"
	"github.com/rin2yh/study-architecture/server/order/internal/client/product"
)

var ErrProductNotFound = errors.New("product not found")

// ErrInsufficientStock は予約が在庫不足で拒否された (409)。checkout は致命扱いで 409 を返す (ADR-[[202606262000]])。
var ErrInsufficientStock = errors.New("insufficient stock")

var ErrUpstream = errors.New("upstream service error")

type ProductSnapshot struct {
	ID             int64
	Name           string
	UnitPriceCents int64
}

type ProductPort interface {
	FetchProduct(ctx context.Context, id int64) (ProductSnapshot, error)
}

type PaymentPort interface {
	CreatePayment(ctx context.Context, orderID, amountCents int64, method string) (int64, error)
}

type ReserveLine struct {
	ProductID int64
	Quantity  int32
}

type InventoryPort interface {
	Reserve(ctx context.Context, orderID int64, lines []ReserveLine) error
	Release(ctx context.Context, orderID int64) error
}

type ProductClient struct {
	c product.ClientWithResponsesInterface
}

var _ ProductPort = (*ProductClient)(nil)

func NewProductClient() (*ProductClient, error) {
	base := os.Getenv("PRODUCT_API_URL")
	if base == "" {
		return nil, errors.New("PRODUCT_API_URL is required")
	}
	c, err := product.NewClientWithResponses(base, product.WithHTTPClient(httpx.NewResilientClient("order->product")))
	if err != nil {
		return nil, err
	}
	return &ProductClient{c: c}, nil
}

func (p *ProductClient) FetchProduct(ctx context.Context, id int64) (ProductSnapshot, error) {
	res, err := p.c.GetProductWithResponse(ctx, id)
	if err != nil {
		return ProductSnapshot{}, fmt.Errorf("%w: get product %d: %v", ErrUpstream, id, err)
	}
	if res.StatusCode() == http.StatusNotFound {
		return ProductSnapshot{}, fmt.Errorf("%w: id %d", ErrProductNotFound, id)
	}
	if res.JSON200 == nil {
		return ProductSnapshot{}, fmt.Errorf("%w: get product %d returned %d", ErrUpstream, id, res.StatusCode())
	}
	return ProductSnapshot{ID: res.JSON200.Id, Name: res.JSON200.Name, UnitPriceCents: res.JSON200.PriceCents}, nil
}

type PaymentClient struct {
	c payment.ClientWithResponsesInterface
}

var _ PaymentPort = (*PaymentClient)(nil)

func NewPaymentClient() (*PaymentClient, error) {
	base := os.Getenv("PAYMENT_API_URL")
	if base == "" {
		return nil, errors.New("PAYMENT_API_URL is required")
	}
	c, err := payment.NewClientWithResponses(base, payment.WithHTTPClient(httpx.NewResilientClient("order->payment")))
	if err != nil {
		return nil, err
	}
	return &PaymentClient{c: c}, nil
}

func (p *PaymentClient) CreatePayment(ctx context.Context, orderID, amountCents int64, method string) (int64, error) {
	res, err := p.c.CreatePaymentWithResponse(ctx, payment.CreatePaymentJSONRequestBody{
		OrderId:     orderID,
		AmountCents: amountCents,
		Method:      method,
		Status:      "pending",
	})
	if err != nil {
		return 0, fmt.Errorf("%w: create payment for order %d: %v", ErrUpstream, orderID, err)
	}
	if res.JSON201 == nil {
		return 0, fmt.Errorf("%w: create payment for order %d returned %d", ErrUpstream, orderID, res.StatusCode())
	}
	return res.JSON201.Id, nil
}

type InventoryClient struct {
	c inventory.ClientWithResponsesInterface
}

var _ InventoryPort = (*InventoryClient)(nil)

func NewInventoryClient() (*InventoryClient, error) {
	base := os.Getenv("INVENTORY_API_URL")
	if base == "" {
		return nil, errors.New("INVENTORY_API_URL is required")
	}
	c, err := inventory.NewClientWithResponses(base, inventory.WithHTTPClient(httpx.NewResilientClient("order->inventory")))
	if err != nil {
		return nil, err
	}
	return &InventoryClient{c: c}, nil
}

func (i *InventoryClient) Reserve(ctx context.Context, orderID int64, lines []ReserveLine) error {
	body := inventory.ReserveJSONRequestBody{OrderId: orderID}
	body.Lines = make([]inventory.ReserveLine, 0, len(lines))
	for _, l := range lines {
		body.Lines = append(body.Lines, inventory.ReserveLine{ProductId: l.ProductID, Quantity: int(l.Quantity)})
	}
	res, err := i.c.ReserveWithResponse(ctx, body)
	if err != nil {
		return fmt.Errorf("%w: reserve for order %d: %v", ErrUpstream, orderID, err)
	}
	if res.StatusCode() == http.StatusConflict {
		return fmt.Errorf("%w: order %d", ErrInsufficientStock, orderID)
	}
	if res.JSON201 == nil {
		return fmt.Errorf("%w: reserve for order %d returned %d", ErrUpstream, orderID, res.StatusCode())
	}
	return nil
}

func (i *InventoryClient) Release(ctx context.Context, orderID int64) error {
	res, err := i.c.ReleaseReservationWithResponse(ctx, orderID)
	if err != nil {
		return fmt.Errorf("%w: release for order %d: %v", ErrUpstream, orderID, err)
	}
	if res.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("%w: release for order %d returned %d", ErrUpstream, orderID, res.StatusCode())
	}
	return nil
}
