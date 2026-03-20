package payment

import (
	"context"

	"example.com/minimal/internal/order"
)

// DummyGateway is a payment adapter that always succeeds.
// Replace this with a real Stripe/Alipay/etc. adapter in production.
type DummyGateway struct{}

func NewDummyGateway() *DummyGateway { return &DummyGateway{} }

func (g *DummyGateway) Charge(_ context.Context, _ string, _ int64) error {
	// TODO: call real payment provider SDK here, map SDK errors → domain errors.
	return nil
}

// compile-time check that DummyGateway satisfies the port.
var _ order.PaymentGateway = (*DummyGateway)(nil)
