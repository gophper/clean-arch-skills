package order_test

import (
	"context"
	"testing"
	"time"

	"example.com/minimal/internal/order"
)

// ---------------------------------------------------------------------------
// Fake adapters — implement Ports using in-memory state, no DB / HTTP / MQ.
// ---------------------------------------------------------------------------

type fakeRepo struct{ saved *order.Order }

func (r *fakeRepo) Save(_ context.Context, o *order.Order) error {
	cp := *o
	r.saved = &cp
	return nil
}
func (r *fakeRepo) FindByID(_ context.Context, id string) (*order.Order, error) {
	if r.saved != nil && r.saved.ID == id {
		cp := *r.saved
		return &cp, nil
	}
	return nil, order.ErrNotFound
}

type alwaysOKPayment struct{}

func (alwaysOKPayment) Charge(_ context.Context, _ string, _ int64) error { return nil }

type failPayment struct{}

func (failPayment) Charge(_ context.Context, _ string, _ int64) error {
	return order.ErrPaymentFailed
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

type fixedID struct{ id string }

func (g fixedID) NewID() string { return g.id }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateOrder_HappyPath(t *testing.T) {
	repo := &fakeRepo{}
	now := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	uc := order.NewCreateOrderUseCase(
		repo,
		alwaysOKPayment{},
		fixedClock{t: now},
		fixedID{id: "order-123"},
	)

	res, err := uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "order-123" {
		t.Errorf("expected ID=order-123, got %s", res.ID)
	}
	if repo.saved == nil {
		t.Fatal("expected order to be saved in repo")
	}
	if !repo.saved.CreatedAt.Equal(now) {
		t.Errorf("createdAt mismatch: want %v got %v", now, repo.saved.CreatedAt)
	}
}

func TestCreateOrder_InvalidAmount(t *testing.T) {
	uc := order.NewCreateOrderUseCase(&fakeRepo{}, alwaysOKPayment{}, fixedClock{}, fixedID{})
	_, err := uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: 0})
	if err != order.ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateOrder_NegativeAmount(t *testing.T) {
	uc := order.NewCreateOrderUseCase(&fakeRepo{}, alwaysOKPayment{}, fixedClock{}, fixedID{})
	_, err := uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: -1})
	if err != order.ErrInvalidAmount {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestCreateOrder_PaymentFailed(t *testing.T) {
	uc := order.NewCreateOrderUseCase(&fakeRepo{}, failPayment{}, fixedClock{}, fixedID{})
	_, err := uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: 100})
	if err != order.ErrPaymentFailed {
		t.Errorf("expected ErrPaymentFailed, got %v", err)
	}
}

func TestCreateOrder_RepoNotSavedOnPaymentFailure(t *testing.T) {
	repo := &fakeRepo{}
	uc := order.NewCreateOrderUseCase(repo, failPayment{}, fixedClock{}, fixedID{})
	_, _ = uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: 100})
	if repo.saved != nil {
		t.Error("repo should NOT have saved the order when payment fails")
	}
}
