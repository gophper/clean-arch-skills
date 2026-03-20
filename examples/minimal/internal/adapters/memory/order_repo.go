package memory

import (
	"context"
	"sync"

	"example.com/minimal/internal/order"
)

// OrderRepo is an in-memory implementation of order.OrderRepository.
// It is used in tests and as the default adapter in the minimal demo.
type OrderRepo struct {
	mu    sync.RWMutex
	store map[string]*order.Order
}

// NewOrderRepo returns a ready-to-use in-memory order repository.
func NewOrderRepo() *OrderRepo {
	return &OrderRepo{store: make(map[string]*order.Order)}
}

func (r *OrderRepo) Save(_ context.Context, o *order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.store[o.ID]; exists {
		return order.ErrConflict
	}
	cp := *o
	r.store[o.ID] = &cp
	return nil
}

func (r *OrderRepo) FindByID(_ context.Context, id string) (*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if o, ok := r.store[id]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, order.ErrNotFound
}

// compile-time check that OrderRepo satisfies the port.
var _ order.OrderRepository = (*OrderRepo)(nil)
