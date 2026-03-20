package order

import (
	"context"
	"time"
)

// OrderRepository is the persistence port defined by the business side.
// Adapters in the infrastructure layer implement this interface.
type OrderRepository interface {
	Save(ctx context.Context, o *Order) error
	FindByID(ctx context.Context, id string) (*Order, error)
}

// PaymentGateway is the payment port. Any payment provider adapter implements this.
type PaymentGateway interface {
	Charge(ctx context.Context, userID string, amount int64) error
}

// Clock abstracts "wall clock time" so tests can use a fixed, deterministic time.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts ID / UUID generation for reproducible tests.
type IDGenerator interface {
	NewID() string
}
