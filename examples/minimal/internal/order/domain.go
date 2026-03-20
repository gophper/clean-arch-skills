package order

import (
	"errors"
	"time"
)

// Domain errors — adapters map external errors to these; handlers map these to HTTP status codes.
var (
	ErrNotFound      = errors.New("order: not found")
	ErrConflict      = errors.New("order: conflict")
	ErrInvalidAmount = errors.New("order: invalid amount")
	ErrPaymentFailed = errors.New("order: payment failed")
)

// Order is the domain entity. It uses plain Go types — no ORM tags, no framework types.
type Order struct {
	ID        string
	UserID    string
	Amount    int64
	CreatedAt time.Time
}

// CreateOrderCommand is the stable input contract for the CreateOrder use case.
type CreateOrderCommand struct {
	UserID string
	Amount int64
}

// CreateOrderResult is the stable output contract — never exposes internal models.
type CreateOrderResult struct {
	ID        string
	UserID    string
	Amount    int64
	CreatedAt time.Time
}
