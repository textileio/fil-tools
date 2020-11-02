package client

import (
	"context"
	"math/big"

	proto "github.com/textileio/powergate/proto/powergate/v1"
)

// Wallet provides an API for managing filecoin wallets.
type Wallet struct {
	client proto.PowergateServiceClient
}

// NewAddressOption is a function that changes a NewAddressConfig.
type NewAddressOption func(r *proto.NewAddressRequest)

// WithMakeDefault specifies if the new address should become the default.
func WithMakeDefault(makeDefault bool) NewAddressOption {
	return func(r *proto.NewAddressRequest) {
		r.MakeDefault = makeDefault
	}
}

// WithAddressType specifies the type of address to create.
func WithAddressType(addressType string) NewAddressOption {
	return func(r *proto.NewAddressRequest) {
		r.AddressType = addressType
	}
}

// Balance gets a filecoin wallet's balance.
func (w *Wallet) Balance(ctx context.Context, address string) (*proto.BalanceResponse, error) {
	return w.client.Balance(ctx, &proto.BalanceRequest{Address: address})
}

// NewAddress created a new wallet address managed by the storage profile.
func (w *Wallet) NewAddress(ctx context.Context, name string, options ...NewAddressOption) (*proto.NewAddressResponse, error) {
	r := &proto.NewAddressRequest{Name: name}
	for _, opt := range options {
		opt(r)
	}
	return w.client.NewAddress(ctx, r)
}

// Addresses returns a list of addresses managed by the storage profile.
func (w *Wallet) Addresses(ctx context.Context) (*proto.AddressesResponse, error) {
	return w.client.Addresses(ctx, &proto.AddressesRequest{})
}

// SendFil sends fil from a managed address to any another address, returns immediately but funds are sent asynchronously.
func (w *Wallet) SendFil(ctx context.Context, from string, to string, amount *big.Int) (*proto.SendFilResponse, error) {
	req := &proto.SendFilRequest{
		From:   from,
		To:     to,
		Amount: amount.String(),
	}
	return w.client.SendFil(ctx, req)
}

// SignMessage signs a message with a stprage profile wallet address.
func (w *Wallet) SignMessage(ctx context.Context, address string, message []byte) (*proto.SignMessageResponse, error) {
	r := &proto.SignMessageRequest{Address: address, Message: message}
	return w.client.SignMessage(ctx, r)
}

// VerifyMessage verifies a message signature from a wallet address.
func (w *Wallet) VerifyMessage(ctx context.Context, address string, message, signature []byte) (*proto.VerifyMessageResponse, error) {
	r := &proto.VerifyMessageRequest{Address: address, Message: message, Signature: signature}
	return w.client.VerifyMessage(ctx, r)
}
