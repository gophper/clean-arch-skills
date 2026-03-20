# clean-architecture-general

> **目的**：通过可运行脚手架生成"最小 Go demo"，帮助读者快速理解 Clean Architecture（Ports & Adapters）原则；同时提供 AI 工作流，让 AI 在此 demo 基础上按需扩展生成代码，保持依赖方向正确。

---

## Skill Contract

| 项目 | 内容 |
|------|------|
| **适用范围** | Go 项目（Web API / CLI / consumer / batch）中有业务规则 + 有外部依赖的模块 |
| **非目标** | 极小脚本、无业务规则的纯 CRUD、强依赖 ORM 的快速原型 |
| **输入** | 业务用例列表、外部依赖清单（DB/MQ/第三方 SDK/时钟/ID）、入口类型（HTTP/CLI/consumer） |
| **输出** | domain 错误 + Port 接口 + UseCase + Adapter（≥2）+ 入口 handler + 组合根 + 单元测试 |
| **验收标准 (DoD)** | ① domain/usecase 包不 import gin/gorm/kafka/任何外部框架；② adapter 把外部错误映射为 domain errors；③ 用例单测不依赖真实 DB/MQ；④ 组合根是唯一 new 跨层依赖的地方 |

---

## 脚手架快速开始

> **三条命令，从零到运行**

```bash
# 1. 生成（或重新生成）最小 demo — 幂等，可重复执行
make scaffold

# 2. 启动 demo 服务（监听 :8080）
make run

# 3. 运行所有单元测试（无需 DB / MQ）
make test
```

手动等价命令（无 make）：

```bash
go run ./cmd/scaffold                         # 生成 examples/minimal/
cd examples/minimal && go run ./cmd/demo      # 运行
cd examples/minimal && go test ./... -v       # 测试
```

验证运行：

```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"userId":"u1","amount":100}' | jq .
```

---

## 最小 Demo 目录结构

```
examples/minimal/
├── go.mod                                    # 独立 Go 模块，零外部依赖（仅 stdlib）
├── cmd/demo/main.go                          # 组合根 — 唯一允许跨层 new 的地方
└── internal/
    ├── order/                                # 业务核心（domain layer）
    │   ├── domain.go                         #   Order 实体 + 领域错误
    │   ├── ports.go                          #   Port 接口（OrderRepository / PaymentGateway / Clock / IDGenerator）
    │   ├── usecase.go                        #   CreateOrderUseCase（业务编排）
    │   └── usecase_test.go                   #   用例单元测试（使用 fake adapter，无 DB）
    ├── adapters/                             # 基础设施层（adapters implement ports）
    │   ├── memory/order_repo.go              #   内存 OrderRepository（demo & 测试用）
    │   ├── system/clock.go                   #   SystemClock（实际时间）
    │   ├── system/idgen.go                   #   crypto/rand IDGenerator
    │   └── payment/dummy.go                  #   Dummy PaymentGateway（始终成功）
    └── httpapi/handler.go                    # 入口层（thin net/http handler）
```

依赖方向（单向，不可反转）：

```
httpapi ──► order (domain + ports + usecase)
adapters ──► order (adapters implement ports)
cmd/demo ──► all (仅组合根知道所有层)
```

---

## 示例代码片段索引

### 1. 领域层：实体 + 错误（`internal/order/domain.go`）

```go
package order

import (
    "errors"
    "time"
)

var (
    ErrNotFound      = errors.New("order: not found")
    ErrConflict      = errors.New("order: conflict")
    ErrInvalidAmount = errors.New("order: invalid amount")
    ErrPaymentFailed = errors.New("order: payment failed")
)

type Order struct {
    ID        string
    UserID    string
    Amount    int64
    CreatedAt time.Time
}

// 稳定边界契约：Command/Result 不暴露 ORM 模型或框架类型
type CreateOrderCommand struct{ UserID string; Amount int64 }
type CreateOrderResult  struct{ ID string; UserID string; Amount int64; CreatedAt time.Time }
```

### 2. Ports（`internal/order/ports.go`）

```go
package order

import (
    "context"
    "time"
)

// 持久化端口 — 由 adapter 实现，usecase 只知道这个接口
type OrderRepository interface {
    Save(ctx context.Context, o *Order) error
    FindByID(ctx context.Context, id string) (*Order, error)
}

// 支付端口 — 任何支付提供商的 adapter 实现此接口
type PaymentGateway interface {
    Charge(ctx context.Context, userID string, amount int64) error
}

// 外部性端口：时间与 ID（确保测试可复现）
type Clock       interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
```

### 3. 用例（`internal/order/usecase.go`）

```go
package order

import "context"

// 用例只依赖 Port（接口），不依赖 gorm / gin / kafka 等任何外部库
type CreateOrderUseCase struct {
    repo  OrderRepository
    pay   PaymentGateway
    clock Clock
    idGen IDGenerator
}

func NewCreateOrderUseCase(repo OrderRepository, pay PaymentGateway, clock Clock, idGen IDGenerator) *CreateOrderUseCase {
    return &CreateOrderUseCase{repo: repo, pay: pay, clock: clock, idGen: idGen}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, cmd CreateOrderCommand) (*CreateOrderResult, error) {
    if cmd.Amount <= 0 {
        return nil, ErrInvalidAmount // 业务规则在用例里
    }
    o := &Order{ID: uc.idGen.NewID(), UserID: cmd.UserID, Amount: cmd.Amount, CreatedAt: uc.clock.Now()}
    if err := uc.pay.Charge(ctx, cmd.UserID, cmd.Amount); err != nil {
        return nil, ErrPaymentFailed
    }
    if err := uc.repo.Save(ctx, o); err != nil {
        return nil, err
    }
    return &CreateOrderResult{ID: o.ID, UserID: o.UserID, Amount: o.Amount, CreatedAt: o.CreatedAt}, nil
}
```

### 4. Adapter — 内存 Repo（`internal/adapters/memory/order_repo.go`）

```go
package memory

import (
    "context"
    "sync"
    "example.com/minimal/internal/order"
)

type OrderRepo struct {
    mu    sync.RWMutex
    store map[string]*order.Order
}

func NewOrderRepo() *OrderRepo { return &OrderRepo{store: make(map[string]*order.Order)} }

func (r *OrderRepo) Save(_ context.Context, o *order.Order) error {
    r.mu.Lock(); defer r.mu.Unlock()
    if _, exists := r.store[o.ID]; exists {
        return order.ErrConflict // ✅ 外部错误 → 领域错误
    }
    cp := *o; r.store[o.ID] = &cp
    return nil
}

var _ order.OrderRepository = (*OrderRepo)(nil) // 编译期接口检查
```

### 5. HTTP Handler — 薄入口（`internal/httpapi/handler.go`）

```go
package httpapi

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "time"
    "example.com/minimal/internal/order"
)

// handler 依赖接口，不依赖具体 usecase 类型
type createOrderUseCase interface {
    Execute(ctx context.Context, cmd order.CreateOrderCommand) (*order.CreateOrderResult, error)
}

func (h *OrderHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
    var req createOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest); return
    }
    res, err := h.createUC.Execute(r.Context(), order.CreateOrderCommand{UserID: req.UserID, Amount: req.Amount})
    if err != nil {
        // ✅ handler 只识别领域错误，不判断 gorm/SDK 错误
        switch {
        case errors.Is(err, order.ErrInvalidAmount): http.Error(w, err.Error(), http.StatusBadRequest)
        case errors.Is(err, order.ErrConflict):      http.Error(w, err.Error(), http.StatusConflict)
        case errors.Is(err, order.ErrPaymentFailed): http.Error(w, err.Error(), http.StatusPaymentRequired)
        default:                                     http.Error(w, "internal server error", http.StatusInternalServerError)
        }
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(res)
}
```

### 6. 组合根（`cmd/demo/main.go`）

```go
package main

import (
    "log"
    "net/http"
    // 组合根是唯一允许 import 所有层的地方
    "example.com/minimal/internal/adapters/memory"
    "example.com/minimal/internal/adapters/payment"
    "example.com/minimal/internal/adapters/system"
    "example.com/minimal/internal/httpapi"
    "example.com/minimal/internal/order"
)

func main() {
    repo  := memory.NewOrderRepo()        // adapter 实现 port
    pay   := payment.NewDummyGateway()
    clock := system.NewClock()
    idGen := system.NewIDGenerator()

    createOrderUC := order.NewCreateOrderUseCase(repo, pay, clock, idGen) // 显式注入

    mux := http.NewServeMux()
    httpapi.NewOrderHandler(createOrderUC).RegisterRoutes(mux)
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### 7. 用例单元测试（`internal/order/usecase_test.go`）

```go
package order_test

// Fake adapter — 无 DB / HTTP / MQ
type fakeRepo struct{ saved *order.Order }
func (r *fakeRepo) Save(_ context.Context, o *order.Order) error { cp := *o; r.saved = &cp; return nil }
func (r *fakeRepo) FindByID(_ context.Context, id string) (*order.Order, error) { /* ... */ }

type fixedClock struct{ t time.Time }
func (c fixedClock) Now() time.Time { return c.t }

func TestCreateOrder_HappyPath(t *testing.T) {
    repo := &fakeRepo{}
    uc   := order.NewCreateOrderUseCase(repo, alwaysOKPayment{}, fixedClock{t: now}, fixedID{id: "order-123"})
    res, err := uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: 100})
    // assert res.ID == "order-123", repo.saved != nil ...
}
```

---

## 执行清单（可勾选 SOP）

每次新增业务功能时，按顺序完成以下步骤：

### Step A ✅ 识别边界
- [ ] 写清楚：入口（HTTP/CLI/consumer/cron）、用例名称、外部依赖清单
- **产物**：一行描述，如 "入口: POST /orders；用例: CreateOrder；依赖: OrderRepo, PaymentGateway, Clock, IDGenerator"
- **验收**：无遗漏外部依赖

### Step B ✅ 定义领域错误（`internal/<domain>/domain.go`）
- [ ] 用 `errors.New` 定义所有业务错误 sentinel
- [ ] 定义 Command / Result DTO（不包含 ORM tag / 框架类型）
- **产物**：`domain.go` 中的 `var ErrXxx = errors.New(...)` 和 Command/Result 结构体
- **验收**：文件无任何外部 import（只有 `errors`、`time` 等 stdlib）

### Step C ✅ 声明 Ports（`internal/<domain>/ports.go`）
- [ ] 按用例需要拆小接口（ISP 原则），不要大而全
- [ ] 接口以业务语言命名（`OrderRepository`, `PaymentGateway`，避免 `Manager/Helper`）
- **产物**：`ports.go` 中的 interface 声明
- **验收**：ports.go 只有 stdlib import；usecase 能引用这些接口编译通过

### Step D ✅ 实现用例（`internal/<domain>/usecase.go`）
- [ ] 构造函数显式注入所有 Port（含 Clock / IDGenerator）
- [ ] 业务规则（校验/状态机/编排/幂等）全部在 Execute() 里
- **产物**：`NewXxxUseCase(...)` 构造函数 + `Execute(ctx, cmd)` 方法
- **失败信号**：usecase.go 中出现 `gorm` / `gin` / `kafka` / `http.Request` 等 import → 立即移除

### Step E ✅ 实现 Adapters（`internal/adapters/<name>/`）
- [ ] 每个 adapter 实现对应 Port
- [ ] adapter 内完成：调用外部库 + 数据转换（ORM model ↔ domain model）+ **错误映射**（外部错误 → domain error）
- [ ] 添加编译期接口检查：`var _ order.OrderRepository = (*GormOrderRepo)(nil)`
- **产物**：`*_adapter.go` 或 `*_repo.go`，`var _ Port = (*Adapter)(nil)`
- **验收**：domain/usecase 包编译时不依赖此 adapter 包

### Step F ✅ 实现入口层（`internal/httpapi/` 或 `cmd/`）
- [ ] handler 只做：decode request → validate → call usecase → map response / map domain errors
- [ ] 用 `errors.Is` 判断 domain error，映射 HTTP 状态码；不判断 gorm/SDK 错误
- **产物**：薄 handler，行数 < 50 行（不含 DTO 定义）
- **失败信号**：handler 里出现 `db.Transaction` / `repo.Save` / `sdk.Charge` → 移入用例

### Step G ✅ 更新组合根（`cmd/<app>/main.go`）
- [ ] 只在 main.go（或 wire.go）中 new 所有 adapter，注入到 usecase，注入到 handler
- [ ] 不在任何业务/适配器代码中 new 依赖
- **产物**：main.go 新增对应 adapter 初始化与注入行
- **验收**：删除 main.go 后，其他包都能编译

### Step H ✅ 编写用例单元测试
- [ ] 用 fake/memory adapter 测用例（不启动 DB/MQ/HTTP）
- [ ] 覆盖：happy path + 每个业务错误分支
- **产物**：`usecase_test.go`，`go test ./internal/<domain>/` 能独立运行
- **验收**：测试不依赖任何 docker / 外部服务

### Step I ✅ 依赖方向验收（Definition of Done）
- [ ] `domain/usecase` 包无外部框架 import
- [ ] adapter 层把所有外部错误转换为 domain errors
- [ ] 单测不依赖真实 DB / MQ
- [ ] 组合根是唯一跨层 new 的地方

---

## 通用原则参考

> 下列原则是 SOP 步骤背后的"为什么"。每条对应 SOP 的一个 Step。

### P1：依赖只能朝向"更稳定的业务核心"
业务核心（用例/领域）**只能依赖抽象（Port/Interface）**，不能依赖外部实现（框架/DB/SDK）。

**检验问题**：能否在不引入任何外部 client 的情况下编译并用 fake 完成单元测试？

### P2：所有外部依赖都必须被隔离（Port + Adapter）
- Adapter 负责：调用外部 → 数据转换 → **错误语义转换**
- 换 DB/SDK 只改 adapter，不改业务核心

### P3：入口层只做"输入输出映射"
- 入口：解析输入 → 调用用例 → 映射输出
- 业务规则（权限/幂等/状态机/事务边界）在用例里

### P4：依赖必须显式注入，禁止隐式全局状态
- Clock / IDGenerator 也是依赖，必须注入
- 禁止包级变量 / Service Locator

### P5：业务核心必须可测试且可复现
- 用 fake adapter 替代真实外部依赖
- 用固定 clock/idGen 确保测试结果可复现

### P6：边界契约稳定，内部模型不泄漏
- 对外使用 Command / Result / DTO
- 不暴露 ORM 实体、SDK response、框架类型

### P7：错误要业务化，外部错误不穿透
- adapter 处：外部错误 → domain error
- handler 处：domain error → HTTP status code

### P8：一致性由用例定义，事务由基础设施实现
```go
// Port（业务侧表达"需要事务"）
type TxManager interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
// 用例：只声明需要原子性，不关心 SQL/ORM 事务 API
func (uc *UseCase) Execute(ctx context.Context) error {
    return uc.tx.WithinTx(ctx, func(txCtx context.Context) error {
        // repo ops...
        return nil
    })
}
```

### P9：以业务语言命名，避免职责坍塌
- ✅ `CreateOrderUseCase`, `OrderRepository`, `PaymentGateway`
- ❌ `OrderManager`, `BusinessHelper`, `CommonUtil`, `XXXServiceImplV2`

---

## 错误映射规范

| 外部错误 | adapter 映射 | handler 映射 |
|----------|-------------|-------------|
| DB unique violation | `ErrConflict` | `409 Conflict` |
| DB record not found | `ErrNotFound` | `404 Not Found` |
| DB timeout | `ErrTimeout`（或 wrap） | `503 Service Unavailable` |
| 支付余额不足 | `ErrPaymentFailed` | `402 Payment Required` |
| 参数校验失败（业务规则） | `ErrInvalidXxx` | `400 Bad Request` |

---

## 分层测试矩阵

| 层级 | 测试类型 | 允许外部依赖 | 优先级 |
|------|----------|------------|--------|
| 用例单元测试 | fake adapter | ❌ 禁止 | **必须** |
| Adapter 单元测试 | testcontainers / embedded DB | ✅ 可选 | 推荐 |
| Handler 集成测试 | `httptest.NewServer` | ❌ fake adapter | 推荐 |
| E2E 测试 | 真实 DB + HTTP | ✅ 可选 | 可选 |

---

## 常见反模式

| 反模式 | 现象 | 修正方向 |
|--------|------|---------|
| 用例依赖 ORM | `usecase.go` import `gorm.io/gorm` | 定义 Port，把 gorm 放 adapter |
| Handler 做业务编排 | handler 里 `db.Transaction(...)` | 编排逻辑移入用例 |
| 全局包级变量持有 DB | `var DB *gorm.DB` | 构造函数注入 |
| Adapter 错误穿透 | handler 判断 `gorm.ErrRecordNotFound` | adapter 转换为 `ErrNotFound` |
| 直接 `time.Now()` 在用例 | 无法控制测试时间 | 注入 `Clock` Port |
| Port 接口过大 | 一个接口 30 个方法 | 按用例需要拆小（ISP） |

---

## AI 扩展提示词模板

以下提示词以 `examples/minimal/` 为基准，让 AI 扩展生成代码。**约束已内嵌，不需要重复说明规则。**

### 新增用例

```
参考 examples/minimal/ 的结构，在 internal/order/ 下新增用例 CancelOrder：
- Command: CancelOrderCommand{OrderID string, Reason string}
- Result: CancelOrderResult{CancelledAt time.Time}
- 业务规则：订单不存在 → ErrNotFound；已取消 → ErrConflict
- 新增 Port 方法（如需要）：OrderRepository.FindByID / UpdateStatus
- 约束：usecase 不得 import gin/gorm/任何外部框架；adapter 做错误转换；handler 只做 bind/validate/call/map
- 同步更新组合根 cmd/demo/main.go
- 为新用例编写单元测试（fake adapter，不依赖 DB）
```

### 新增 Adapter（替换内存 Repo 为 GORM）

```
参考 examples/minimal/internal/adapters/memory/order_repo.go，
新增 internal/adapters/gorm/order_repo.go 实现 order.OrderRepository：
- Save: 捕获 gorm.ErrDuplicatedKey → 返回 order.ErrConflict；其他错误透传
- FindByID: 捕获 gorm.ErrRecordNotFound → 返回 order.ErrNotFound
- 添加 var _ order.OrderRepository = (*GormOrderRepo)(nil) 编译期检查
- 不改动任何 internal/order/ 文件
```

### 新增 Port

```
在 internal/order/ports.go 中新增 Port：
type NotificationSender interface {
    SendOrderConfirmation(ctx context.Context, userID string, orderID string) error
}
同步更新 CreateOrderUseCase 构造函数注入此 Port，
新增 adapters/notification/dummy.go 实现（直接 return nil），
更新组合根注入，并为新 Port 编写 fake 用于单元测试。
约束：usecase 只依赖此接口，不依赖任何 HTTP/MQ SDK。
```

### 新增错误映射

```
在 internal/order/domain.go 中新增 domain error：
var ErrOrderLimitExceeded = errors.New("order: limit exceeded")

在 adapter 层：当外部 API 返回 http 429 Too Many Requests 时，映射为此错误。
在 handler 层：当遇到此错误时返回 HTTP 429，body: {"error": "order limit exceeded"}。
```

### 新增单元测试

```
参考 examples/minimal/internal/order/usecase_test.go 的 fake adapter 风格，
为 CancelOrderUseCase 新增测试：
- TestCancelOrder_HappyPath
- TestCancelOrder_NotFound（fake repo 返回 ErrNotFound）
- TestCancelOrder_AlreadyCancelled（fake repo 返回 ErrConflict）
不依赖任何真实 DB / HTTP / MQ，使用 fixedClock / fixedID。
```

---

## 从 SOLID 原则看本 Skill

| 原则 | 在本 Skill 的体现 | 反例 |
|------|-----------------|------|
| **S** SRP | 用例编排业务、adapter 对接外部、handler 处理 IO | handler 里做事务/查库/调 SDK |
| **O** OCP | 新增 adapter/用例无需修改业务核心 | 用例里 `switch provider { case A... case B... }` |
| **L** LSP | 所有 adapter 实现 Port 时语义一致（如 NotFound 行为） | 不同实现返回不同的 nil/err 语义 |
| **I** ISP | Port 按用例需要拆小接口 | `type Repository interface { 30 个方法 }` |
| **D** DIP | 用例依赖抽象 Port，不依赖 gorm/gin | 用例直接 `time.Now()` / `uuid.New()` |

---

## 注意事项

- 本示例刻意使用 stdlib 零外部依赖，以最小化学习曲线。生产项目可替换 adapter 实现（如 GORM repo、Stripe gateway、Redis cache），**业务核心代码不需要改动**，这正是 Clean Architecture 的价值。
- 事务处理：本 demo 无 DB，因此不包含 TxManager Port。如需事务，参考原则 P8 定义 `TxManager` Port，由 GORM/SQL adapter 实现。
- 本 Skill 约束依赖方向与架构边界，不约束目录结构命名（只要依赖方向正确即可）。
