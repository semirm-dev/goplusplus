package transport

import "github.com/semirm-dev/goplusplus/examples/a6-a7-project-layout/discount"

// Other features use discount through this client only.
// If discount moves to its own service, this becomes an HTTP/gRPC client.
type Client struct{}

func (Client) Percent(code string) int64 {
	return discount.Percent(code)
}
