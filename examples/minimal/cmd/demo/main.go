package main

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
