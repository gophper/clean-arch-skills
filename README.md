# clean-arch-skills

A runnable scaffold that generates a **minimal Go demo** of Clean Architecture (Ports & Adapters), suitable for:

- 🎓 **Learning** the architecture principles hands-on by reading and running real code.
- 🤖 **AI-assisted development** — use the demo as a stable baseline for generating new use cases, adapters, handlers and tests that obey the dependency rules.

See **[SKILL.md](./SKILL.md)** for the full skill document, including principles, checklist, Definition of Done and AI prompt templates.

---

## Quick start — three commands

```bash
# 1. Generate (or regenerate) the minimal demo
make scaffold

# 2. Run the demo server  (listens on :8080)
make run

# 3. Run all unit tests  (no DB / MQ required)
make test
```

### Try it

```bash
# In a second terminal while `make run` is running:
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"userId":"u1","amount":100}' | jq .
```

Expected response (`201 Created`):

```json
{
  "id": "3a9f...",
  "userId": "u1",
  "amount": 100,
  "createdAt": "2026-03-20T11:41:09Z"
}
```

---

## Generated demo layout

```
examples/minimal/
├── go.mod                                    # standalone module, stdlib only
├── cmd/demo/main.go                          # Composition Root — wires all layers
└── internal/
    ├── order/
    │   ├── domain.go                         # Order entity + domain errors
    │   ├── ports.go                          # OrderRepository, PaymentGateway, Clock, IDGenerator
    │   ├── usecase.go                        # CreateOrderUseCase (business logic)
    │   └── usecase_test.go                   # unit tests with fake adapters
    ├── adapters/
    │   ├── memory/order_repo.go              # in-memory OrderRepository (no DB)
    │   ├── system/clock.go                   # real wall-clock adapter
    │   ├── system/idgen.go                   # crypto/rand ID generator
    │   └── payment/dummy.go                  # always-OK payment adapter
    └── httpapi/handler.go                    # thin net/http handler
```

Dependency direction: `httpapi` → `order` ← `adapters` (adapters implement ports; handler calls use case).

---

## Requirements

- Go 1.21+
- `make` (standard on Linux/macOS; on Windows use `go run ./cmd/scaffold` directly)
