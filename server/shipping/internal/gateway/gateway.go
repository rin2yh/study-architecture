// Package gateway は shipping が order サービスを呼ぶ出力ポートと生成クライアント実装をまとめる。
package gateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/rin2yh/study-architecture/server/internal/httpx/resilience"
	"github.com/rin2yh/study-architecture/server/internal/order"
	orderapi "github.com/rin2yh/study-architecture/server/shipping/internal/client/order"
)

var ErrUpstream = errors.New("upstream service error")

// Destination は注文時点で確定した配送先のスナップショット (ADR-[[202606301000]])。
type Destination struct {
	Recipient  string
	PostalCode string
	Prefecture string
	City       string
	Line1      string
}

type OrderPort interface {
	FetchDestination(ctx context.Context, orderID order.ID) (Destination, error)
}

type OrderClient struct {
	c orderapi.ClientWithResponsesInterface
}

var _ OrderPort = (*OrderClient)(nil)

func NewOrderClient() (*OrderClient, error) {
	base := os.Getenv("ORDER_API_URL")
	if base == "" {
		return nil, errors.New("ORDER_API_URL is required")
	}
	c, err := orderapi.NewClientWithResponses(base, orderapi.WithHTTPClient(resilience.NewClient("shipping->order")))
	if err != nil {
		return nil, err
	}
	return &OrderClient{c: c}, nil
}

func (o *OrderClient) FetchDestination(ctx context.Context, orderID order.ID) (Destination, error) {
	// 生成クライアントが int64 を受けるのはこの層の事情で、ドメインの order.ID は表現を持たない。
	id, err := strconv.ParseInt(orderID.String(), 10, 64)
	if err != nil {
		return Destination{}, fmt.Errorf("%w: invalid order id %q: %v", ErrUpstream, orderID, err)
	}
	res, err := o.c.GetOrderWithResponse(ctx, id)
	if err != nil {
		return Destination{}, fmt.Errorf("%w: get order %s: %v", ErrUpstream, orderID, err)
	}
	if res.JSON200 == nil {
		return Destination{}, fmt.Errorf("%w: get order %s returned %d", ErrUpstream, orderID, res.StatusCode())
	}
	addr := res.JSON200.ShippingAddress
	if addr == nil {
		return Destination{}, nil
	}
	return Destination{
		Recipient:  addr.Recipient,
		PostalCode: addr.PostalCode,
		Prefecture: addr.Prefecture,
		City:       addr.City,
		Line1:      addr.Line1,
	}, nil
}
