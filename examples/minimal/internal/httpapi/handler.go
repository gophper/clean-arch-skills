package httpapi

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
	UserID string `json:"userId"`
	Amount int64  `json:"amount"`
}

type createOrderResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Amount    int64  `json:"amount"`
	CreatedAt string `json:"createdAt"`
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
