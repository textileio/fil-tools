package client

import (
	"context"

	h "github.com/textileio/powergate/health"
	"github.com/textileio/powergate/health/rpc"
)

// Health provides an API for checking node Health
type Health struct {
	client rpc.RPCClient
}

// Check returns the node health status and any related messages
func (health *Health) Check(ctx context.Context) (h.Status, []string, error) {
	resp, err := health.client.Check(ctx, &rpc.CheckRequest{})
	if err != nil {
		return h.Error, nil, err
	}
	status := h.Status(resp.Status)
	return status, resp.Messages, nil
}
