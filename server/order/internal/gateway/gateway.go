// Package gateway は order が他サービス (product / payment) を呼ぶ出力ポートと
// 生成クライアント実装をまとめる。
package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/order/internal/client/payment"
	"github.com/rin2yh/study-architecture/server/order/internal/client/product"
)

var ErrProductNotFound = errors.New("product not found")

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
	CreatePayment(ctx context.Context, orderID, amountCents int64, method, idempotencyKey string) (int64, error)
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
	// 決済作成は idempotency key で冪等なので POST リトライを解禁する (ADR-[[202606261214]])。
	c, err := payment.NewClientWithResponses(base, payment.WithHTTPClient(httpx.NewResilientClient("order->payment", httpx.RetryNonIdempotent())))
	if err != nil {
		return nil, err
	}
	return &PaymentClient{c: c}, nil
}

func (p *PaymentClient) CreatePayment(ctx context.Context, orderID, amountCents int64, method, idempotencyKey string) (int64, error) {
	res, err := p.c.CreatePaymentWithResponse(ctx, payment.CreatePaymentJSONRequestBody{
		OrderId:        orderID,
		AmountCents:    amountCents,
		Method:         method,
		Status:         "pending",
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return 0, fmt.Errorf("%w: create payment for order %d: %v", ErrUpstream, orderID, err)
	}
	if res.JSON201 == nil {
		return 0, fmt.Errorf("%w: create payment for order %d returned %d", ErrUpstream, orderID, res.StatusCode())
	}
	return res.JSON201.Id, nil
}
