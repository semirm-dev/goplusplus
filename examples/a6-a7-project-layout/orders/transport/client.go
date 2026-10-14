package transport

import (
	"context"

	"github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/orders"
)

// For other features. Exposes only what they need.
type Client struct {
	svc *orders.Service
}

func NewClient(svc *orders.Service) *Client {
	return &Client{svc: svc}
}

func (c *Client) Total(ctx context.Context, id string) (int64, error) {
	o, err := c.svc.Get(ctx, id)
	return o.Total, err
}
