package stub

import (
	"google.golang.org/grpc"
)

// Client provides the client api
type Client struct {
	Deals  *Deals
	Wallet *Wallet
}

// NewClient starts the client
func NewClient(target string, opts ...grpc.DialOption) (*Client, error) {
	client := &Client{
		Deals:  &Deals{},
		Wallet: &Wallet{},
	}
	return client, nil
}

// Close closes the client's grpc connection and cancels any active requests
func (c *Client) Close() error {
	return nil
}
