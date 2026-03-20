// scaffold generates (or regenerates) the examples/minimal/ clean-architecture demo.
// Run from the repository root:
//
//	go run ./cmd/scaffold
//
// The generator is idempotent — it overwrites existing files on every run.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root := filepath.Join("examples", "minimal")
	if err := generate(root); err != nil {
		fmt.Fprintf(os.Stderr, "scaffold: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("scaffold: examples/minimal/ generated successfully")
	fmt.Println("")
	fmt.Println("  run:   cd examples/minimal && go run ./cmd/demo")
	fmt.Println("  test:  cd examples/minimal && go test ./...")
}

// generate writes all demo files under the given root directory, overwriting any existing files.
func generate(root string) error {
	for path, content := range files(root) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Printf("  wrote %s\n", path)
	}
	return nil
}

// files returns the path→content map for every file in the minimal demo.
func files(root string) map[string]string {
	j := func(parts ...string) string { return filepath.Join(append([]string{root}, parts...)...) }
	return map[string]string{
		j("go.mod"): goMod,
		j("internal", "order", "domain.go"):                     domainGo,
		j("internal", "order", "ports.go"):                      portsGo,
		j("internal", "order", "usecase.go"):                    usecaseGo,
		j("internal", "order", "usecase_test.go"):               usecaseTestGo,
		j("internal", "adapters", "memory", "order_repo.go"):    memoryRepoGo,
		j("internal", "adapters", "system", "clock.go"):         clockGo,
		j("internal", "adapters", "system", "idgen.go"):         idgenGo,
		j("internal", "adapters", "payment", "dummy.go"):        dummyGo,
		j("internal", "httpapi", "handler.go"):                  handlerGo,
		j("cmd", "demo", "main.go"):                             mainGo,
	}
}

// ---------------------------------------------------------------------------
// File contents
// ---------------------------------------------------------------------------

const goMod = `module example.com/minimal

go 1.21
`

const domainGo = `package order

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
`

const portsGo = `package order

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
`

const usecaseGo = `package order

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
`

const usecaseTestGo = `package order_test

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
`

const memoryRepoGo = `package memory

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
`

const clockGo = `package system

import "time"

// Clock is an adapter that returns the real wall-clock time.
// In tests, swap it with a fake that returns a fixed time.
type Clock struct{}

func NewClock() Clock        { return Clock{} }
func (Clock) Now() time.Time { return time.Now() }
`

const idgenGo = `package system

import (
	"crypto/rand"
	"encoding/hex"
)

// IDGenerator produces random 128-bit (hex-encoded) identifiers using crypto/rand.
// No external dependencies required — stdlib only.
type IDGenerator struct{}

func NewIDGenerator() IDGenerator { return IDGenerator{} }

func (IDGenerator) NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
`

const dummyGo = `package payment

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
`

const handlerGo = `package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"example.com/minimal/internal/order"
)

// createOrderUseCase is the port that the handler depends on.
// This keeps the handler decoupled from the concrete use case type.
type createOrderUseCase interface {
	Execute(ctx context.Context, cmd order.CreateOrderCommand) (*order.CreateOrderResult, error)
}

// OrderHandler is the thin HTTP handler for order-related endpoints.
// Responsibilities: decode request → call use case → encode response / map errors.
// Business logic MUST NOT live here.
type OrderHandler struct {
	createUC createOrderUseCase
}

func NewOrderHandler(createUC createOrderUseCase) *OrderHandler {
	return &OrderHandler{createUC: createUC}
}

// RegisterRoutes attaches all order routes to the given mux.
func (h *OrderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/orders", h.handleCreate)
}

// ---------------------------------------------------------------------------
// Request / Response DTOs — never expose domain entities or adapters here.
// ---------------------------------------------------------------------------

type createOrderRequest struct {
	UserID string ` + "`" + `json:"userId"` + "`" + `
	Amount int64  ` + "`" + `json:"amount"` + "`" + `
}

type createOrderResponse struct {
	ID        string ` + "`" + `json:"id"` + "`" + `
	UserID    string ` + "`" + `json:"userId"` + "`" + `
	Amount    int64  ` + "`" + `json:"amount"` + "`" + `
	CreatedAt string ` + "`" + `json:"createdAt"` + "`" + `
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (h *OrderHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	res, err := h.createUC.Execute(r.Context(), order.CreateOrderCommand{
		UserID: req.UserID,
		Amount: req.Amount,
	})
	if err != nil {
		// Rule 7: handler only knows about domain errors, NOT adapter/DB/SDK errors.
		switch {
		case errors.Is(err, order.ErrInvalidAmount):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, order.ErrConflict):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, order.ErrPaymentFailed):
			http.Error(w, err.Error(), http.StatusPaymentRequired)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createOrderResponse{
		ID:        res.ID,
		UserID:    res.UserID,
		Amount:    res.Amount,
		CreatedAt: res.CreatedAt.Format(time.RFC3339),
	})
}
`

const mainGo = `package main

import (
	"log"
	"net/http"

	"example.com/minimal/internal/adapters/memory"
	"example.com/minimal/internal/adapters/payment"
	"example.com/minimal/internal/adapters/system"
	"example.com/minimal/internal/httpapi"
	"example.com/minimal/internal/order"
)

// main is the Composition Root — the ONLY place allowed to wire all layers together.
// It knows about every adapter and every use case.
func main() {
	// 1. Infrastructure adapters (implement Ports defined in the domain).
	repo  := memory.NewOrderRepo()
	pay   := payment.NewDummyGateway()
	clock := system.NewClock()
	idGen := system.NewIDGenerator()

	// 2. Use cases — inject all dependencies explicitly; no globals, no service locator.
	createOrderUC := order.NewCreateOrderUseCase(repo, pay, clock, idGen)

	// 3. Entry point — thin handler wired to the use case.
	mux := http.NewServeMux()
	h := httpapi.NewOrderHandler(createOrderUC)
	h.RegisterRoutes(mux)

	log.Println("demo server listening on :8080")
	log.Println("try: curl -s -X POST http://localhost:8080/orders -d '{\"userId\":\"u1\",\"amount\":100}' | jq .")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
`
