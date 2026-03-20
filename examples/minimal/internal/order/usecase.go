package order

import "context"

// CreateOrderUseCase orchestrates the CreateOrder business flow.
// It depends ONLY on Ports (interfaces) — never on adapters, frameworks, or databases.
type CreateOrderUseCase struct {
	repo  OrderRepository
	pay   PaymentGateway
	clock Clock
	idGen IDGenerator
}

// NewCreateOrderUseCase is the constructor; all dependencies are injected explicitly.
func NewCreateOrderUseCase(
	repo OrderRepository,
	pay PaymentGateway,
	clock Clock,
	idGen IDGenerator,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{repo: repo, pay: pay, clock: clock, idGen: idGen}
}

// Execute runs the use case. Business rules live here, NOT in the handler or adapter.
func (uc *CreateOrderUseCase) Execute(ctx context.Context, cmd CreateOrderCommand) (*CreateOrderResult, error) {
	// Business rule: amount must be positive.
	if cmd.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	o := &Order{
		ID:        uc.idGen.NewID(),
		UserID:    cmd.UserID,
		Amount:    cmd.Amount,
		CreatedAt: uc.clock.Now(),
	}

	// Orchestration: call payment first, then persist.
	if err := uc.pay.Charge(ctx, cmd.UserID, cmd.Amount); err != nil {
		return nil, ErrPaymentFailed
	}
	if err := uc.repo.Save(ctx, o); err != nil {
		return nil, err
	}

	return &CreateOrderResult{
		ID:        o.ID,
		UserID:    o.UserID,
		Amount:    o.Amount,
		CreatedAt: o.CreatedAt,
	}, nil
}
