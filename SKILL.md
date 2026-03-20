
# clean-architecture-general

## 概览

> 目的：把简洁架构（Clean Architecture）的“规矩”固化成可执行的技能（Skill），用于约束 AI Coding 输出的**架构边界与依赖方向**、开发自检、Code Review，以及后续自动化规则（lint/CI）落地。

本技能**不绑定目录结构**，只约束“依赖方向、边界契约、适配器职责、显式依赖注入、错误边界、事务边界与可测试性”，不仅适用于在线类型服务（如 Web API），也适用于离线批处理（batch）、CLI 工具、消息消费者（consumer）、定时任务（cron/job）等一切“有业务规则 + 有外部依赖”的程序。

### 术语（Rule 0：先统一术语，避免误解）
-  **用例（Use Case）**：业务核心的实现，负责编排业务流程、表达业务规则（权限/幂等/状态机/调用顺序/事务边界等），只依赖 Port，不依赖任何外部实现细节。
- **Port（端口/抽象）**：业务侧定义的抽象接口（interface/protocol），表达业务需要什么能力，例如：
    - `UserRepository`、`PaymentGateway`、`Clock`、`IDGenerator`、`TxManager/UnitOfWork`
- **Adapter（适配器/实现）**：基础设施侧实现 Port 的代码/类型，用来把外部世界（DB/ORM、HTTP SDK、消息队列、缓存、文件系统、系统时间、UUID/随机数）**适配**成 Port 的接口与语义。
    - 之所以叫 *adapter*：因为它承担“接口/数据/语义不匹配时的适配”（Adapter Pattern 的典型职责）
    - 常见 adapter（示例命名）：`GormUserRepoAdapter`、`StripePaymentGatewayAdapter`、`KafkaEventPublisherAdapter`、`SystemClock`、`UUIDGenerator`

> 直观理解：**用例就是各种Service、Provider等、用例只依赖 Port；Adapter 负责对接外部并实现 Port**

---

### 通用原则（不限语言 / 不限服务类型）

> 本节是本技能的“总纲”。这些原则不依赖任何目录结构、不限定编程语言，也不限定系统形态：既适用于 Web API，也适用于离线批处理（batch）、CLI 工具、消息消费者（consumer）、定时任务（cron/job）等一切“有业务规则 + 有外部依赖”的程序。

> 说明：下列原则与后文的执行步骤（Step A ~ Step J）一一对应：P1≈StepB，P2≈StepC，P3≈StepD，P4≈StepE，P5≈StepF，P6≈StepG，P7≈StepH，P8≈StepI，P9≈StepJ。

#### 原则 P1：依赖只能朝向“更抽象、更稳定”的业务核心
业务核心（用例/领域规则）**只能依赖抽象（Port/Interface）**，不能依赖外部实现细节（框架、数据库、ORM、第三方 SDK、消息队列 client、缓存 client、文件系统库等）。

**检验问题**：业务核心能否在不引入任何外部 client（DB/HTTP/MQ/SDK/框架）的情况下编译，并通过 fake 实现完成单元测试？

#### 原则 P2：所有外部依赖都必须被适配与隔离（Port + Adapter）
任何外部依赖都应通过 **Port（业务侧抽象接口）**与 **Adapter（基础设施侧实现）**隔离：
- Adapter 负责对接外部，并完成**数据形状转换**与**语义/错误转换**
- 外部实现替换（换 DB/换 SDK/换 MQ/换协议）应主要改 adapter，不改业务核心

#### 原则 P3：入口层只做“输入输出映射”，业务流程编排在用例里
无论入口是 HTTP handler、消息消费函数、CLI command、定时任务：
- 入口层只做：解析输入/格式校验 → 调用用例 → 映射输出（响应/ack/exit code/日志/指标）
- 权限、幂等、状态机、调用顺序、事务边界、跨依赖编排等业务规则放在用例层

#### 原则 P4：依赖必须显式注入，避免隐式全局状态
业务代码不应从全局单例 / **包级变量（package-level variable）** / Service Locator 取依赖；依赖应通过构造函数或显式参数注入。
包括“外部性”依赖：时间（Clock）、UUID/随机数（IDGenerator/Rand）、环境变量读取、系统调用等。

#### 原则 P5：业务核心必须可测试且可复现
业务核心应能在脱离外部环境（不启动 Web、不连 DB、不接 MQ）的情况下进行单元测试。
做到这一点通常依赖：P1（依赖抽象）+ P2（适配隔离）+ P4（显式注入外部性）。

#### 原则 P6：边界契约稳定，模型不泄漏
对外边界（HTTP/MQ/CLI 输出/文件输出）使用稳定的输入输出契约（Command/Query/Result/DTO），不要泄漏：
- ORM 实体/数据库行结构
- SDK response 结构
- 框架类型（如 web framework 的 context/request/response）

#### 原则 P7：错误要业务化，外部错误不穿透
外部错误（DB error、HTTP 状态码、SDK error code）必须在 adapter 处转换为业务错误（或用例约定的错误类型）。
入口层根据业务错误映射协议层行为（HTTP status、是否重试、是否 ack/nack、exit code 等）。

> 建议：错误要“业务化”，但不等于丢失诊断信息。可在内部用 wrapping/cause/logging 保留原始错误细节，对外仍只暴露业务语义。

#### 原则 P8：一致性由用例定义，事务/一致性机制由基础设施实现
“哪些操作必须原子/一致”属于业务规则，应在用例中表达；
事务细节（SQL tx/ORM tx/分布式事务/Outbox 等）由基础设施实现，并通过 Port（TxManager/UoW/Transaction）承接。

#### 原则 P9：以业务语言命名边界与能力，避免职责坍塌
- 用例/能力应以业务动词命名（Create/Pay/Cancel/...）
- Port 命名表达业务能力（Repository/Gateway/Clock/IDGenerator/...）
- 避免 `Manager/Helper/CommonUtil` 这类泛化命名导致边界失守、职责坍塌

---

## 使用方法

把本技能当作一个“开发与评审的执行流程”。每次做业务变更（新增/改造/修 bug）时，按顺序走一遍：

> 说明：下面正反例以 Go 为主，其他语言/服务形态同理（HTTP/CLI/consumer/job 都适用）。

### Step A：识别边界（入口 / 用例 / 外部依赖）
**要做什么**
- 写清楚本次变更的：
    - 入口（Entry Points）：HTTP handler / CLI command / consumer / job
    - 用例（Use Cases）：Create/Pay/Cancel/...
    - 外部依赖（External Dependencies）：DB/HTTP SDK/MQ/缓存/文件/时间/UUID/随机数等

**正例**
- “入口：POST /orders；用例：CreateOrder；外部依赖：DB(OrderRepo), PaymentGateway, Clock, IDGenerator”

**反例**
- “这次就是改一下接口返回”但没有说明入口/用例/外部依赖，导致后续检查不完整。

---

### Step B：依赖方向检查（Rule 1）
**要做什么**
- 用例/领域代码只能依赖 Port（interface）与业务类型，不能依赖外部实现（ORM/Web 框架/SDK/MQ client/缓存 client）
- 评审问题：**业务核心能否在不引入任何外部 client（DB/HTTP/MQ/SDK/框架）的情况下编译通过？**

**正例**
```go name=examples/stepB/good_usecase_depends_on_ports.go
package order

import "context"

type OrderRepo interface{ Save(context.Context, *Order) error }

type CreateOrderUseCase struct{ repo OrderRepo } // ✅ 只依赖 Port
```

**反例**
```go name=examples/stepB/bad_usecase_depends_on_framework.go
package order

import "github.com/gin-gonic/gin"

// ❌ 用例依赖 Web 框架类型
func Execute(c *gin.Context) error { return nil }
```

---
### Step C：适配器职责（Rule 2）
**要做什么**
把所有外部细节放进 Adapter，并在 Adapter 内完成：
1) 调用外部库/协议（DB/HTTP/MQ/文件/系统调用）
2) 数据形状转换（ORM Model ⇄ 领域模型/DTO）
3) 语义转换（NotFound、HTTP 4xx/5xx、SDK error → 业务错误或 nil/Option）
4) 隔离外部变化（外部替换优先改 adapter，不改用例签名与业务逻辑）

**正例**
```go name=examples/stepC/good_adapter_maps_errors.go
package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"example.com/myapp/internal/order"
)

type GormOrderRepoAdapter struct{ db *gorm.DB }

func (r *GormOrderRepoAdapter) Find(ctx context.Context, id string) (*order.Order, error) {
	var m orderModel
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, order.ErrNotFound // ✅ 外部错误 -> 业务错误/语义
	}
	if err != nil { return nil, err }
	return mapToDomain(m), nil // ✅ ORM model -> domain
}
```

**反例**
```go name=examples/stepC/bad_usecase_handles_gorm_error.go
package order

import (
	"errors"
	"gorm.io/gorm"
)

// ❌ 用例判断 gorm 错误码/语义
func (uc *CreateOrderUseCase) Execute() error {
	if errors.Is(uc.lastErr, gorm.ErrRecordNotFound) { // 不该出现在用例
		return ErrNotFound
	}
	return nil
}
```

---

### Step D：薄入口、厚用例（Rule 3）
**要做什么**
- 入口层只做：输入解析/格式校验 → 调用用例 → 输出映射（响应/ack/exit code）
- 业务流程编排（权限/状态机/幂等/事务边界/跨 repo 协作/调用顺序）必须在用例里
-  厚用例不等于“用例里写一大段代码”，陷入“大泥球”困境，需遵循其他原则（如单一职责、模块边界、命名）来保持可维护性。

**正例**
```go name=examples/stepD/good_handler_calls_usecase.go
package httpapi

// ✅ handler：bind/validate -> call usecase -> map response
func (h *Handler) Create(ctx context.Context, req CreateReq) (CreateResp, error) {
	return h.createUC.Execute(ctx, req.ToCommand())
}
```

**反例**
```go name=examples/stepD/bad_handler_orchestrates_everything.go
package httpapi

// ❌ handler 里做事务/查库/调第三方/状态机编排
func (h *Handler) Create(ctx context.Context, req CreateReq) error {
	h.db.Transaction(func(tx any) error { // 不应出现在入口层
		_ = h.repo.Save(ctx, ...)
		_ = h.paymentSDK.Charge(...)
		return nil
	})
	return nil
}
```

---

### Step E：显式依赖注入（Rule 4）
**要做什么**
- 用例通过构造函数显式注入依赖：Repo/Gateway/Clock/IDGenerator/Tx(UoW)
- 禁止用例内部直接访问全局单例/包级变量/隐式容器
- **Clock/IDGenerator 也算依赖**：因为时间、随机数、UUID 属于“外部性”，需要可控/可复现，便于测试、审计与幂等

**正例**
```go name=examples/stepE/good_constructor_injection.go
package order

type CreateOrderUseCase struct {
	repo  OrderRepository
	clock Clock
	idGen IDGenerator
}

func NewCreateOrderUseCase(repo OrderRepository, clock Clock, idGen IDGenerator) *CreateOrderUseCase {
	return &CreateOrderUseCase{repo: repo, clock: clock, idGen: idGen}
}
```

**反例 1：包级变量（全局状态）持有 DB**
```go name=examples/anti-patterns/bad_package_level_var.go
package dao

import "gorm.io/gorm"

// ❌ 反例：包级变量（全局状态），业务代码可在任何地方直接使用 dao.DB
// 这会导致：依赖隐式、难测试、并发/初始化顺序风险、难以替换实现。
var DB *gorm.DB
```

**反例 2：Service Locator / 容器里“按名字拿依赖”（伪代码）**
```go name=examples/anti-patterns/bad_service_locator.go
package app

// ❌ 反例：隐式依赖（从容器查找），构造函数没有显式声明依赖
// repo := container.MustGet("userRepo").(UserRepo)
```

---

### Step F：可测试性验收（Rule 5）
**要做什么**
- 用例/领域必须能在“不启动 Web、不连 DB、不接 MQ”的情况下做单测
- 用 fake/memory adapter + 可控 clock/idGen 替代真实依赖
- 验收标准：跑用例测试不需要起外部服务

**正例**
```go name=examples/stepF/good_usecase_unit_test.go
package order_test

// ✅ 用 fake repo/clock/idGen 测用例：无需 DB/HTTP
func TestCreateOrder(t *testing.T) { /* ... */ }
```

**反例**
```go name=examples/stepF/bad_test_requires_real_db.go
package order_test

// ❌ 单测必须连真实 DB 才能跑（把用例测试变成集成测试）
func TestCreateOrder(t *testing.T) {
	// start docker-compose, connect mysql, migrate, ...
}
```

---

### Step G：边界契约（Rule 6）
**要做什么**
- 用例输入输出使用稳定的 Command/Result(DTO)
- 不泄漏 ORM 实体、SDK response、框架类型到边界外
- 边界契约稳定（DTO/Command/Result）不等于“到处复制结构体”。只要不泄漏外部模型/框架类型，允许在合适位置复用结构体；是否拆分以团队可维护性为准。

**正例**
```go name=examples/stepG/good_dto_boundary.go
package order

type CreateOrderCommand struct{ UserID string; Amount int64 }
type CreateOrderResult struct{ ID string }
```

**反例**
```go name=examples/stepG/bad_return_orm_model.go
package httpapi

// ❌ HTTP 直接返回 ORM model（泄漏 DB 结构与 tag）
type OrderModel struct {
	ID string `gorm:"primaryKey" json:"id"`
}
```

---

### Step H：错误边界（Rule 7）
**要做什么**
- adapter 把外部错误转换成业务错误（或用例约定的错误类型）
-  内层只根据业务错误类型进行判断，避免和外层实现细节耦合


**正例**
```go name=examples/stepH/good_handler_maps_business_errors.go
func (o *orderRepo) Get(ctx context.Context, id string) (*models.Order, error) {
      var order models.Order
      err := o.WrapSession(o.Table, func(collection *mgo.Collection) error {
		  return collection.Find(bson.M{"id": id}).One(&order)
      })
      if err != nil {
            if errors.Is(err, mgo.ErrNotFound) {
                 return nil, order.ErrNotFound //  禁止返回 mgo.ErrNotFound ！！
            }
            return nil, err
      }
      return &log, nil
}

```

**反例**
```go name=examples/stepH/bad_handler_checks_gorm_error.go

package httpapi

import (
	"errors"
	"gorm.io/gorm"
)

// ❌ handler 判断外部错误（gorm）而不是业务错误
func isNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
```

---

### Step I：一致性与事务（Rule 8）
**要做什么**
- 用例表达一致性边界（哪些步骤必须原子）
- 事务实现细节在基础设施层，通过 Tx Port 承接

**正例**
```go name=examples/stepI/good_tx_port.go
package order

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ✅ 用例：只声明“需要事务”，不关心 ORM 的 Transaction API
func (uc *UseCase) Execute(ctx context.Context) error {
	return uc.tx.WithinTx(ctx, func(txCtx context.Context) error {
		// repo ops...
		return nil
	})
}
```

**反例**
```go name=examples/stepI/bad_usecase_calls_orm_tx.go
package order

import "gorm.io/gorm"

// ❌ 用例直接依赖 ORM 事务 API
func (uc *UseCase) Execute(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error { return nil })
}
```

---

### Step J：命名与模块边界（Rule 9）
**要做什么**
- UseCase/Port/模块以业务语言命名，避免职责坍塌

**正例**
- `CreateOrderUseCase`, `PayOrderUseCase`, `OrderRepository`, `PaymentGateway`

**反例**
- `OrderManager`, `BusinessHelper`, `CommonUtil`, `XXXServiceImplV2`（难表达边界，容易变成“啥都往里塞”）

---

## 示例（golang）

> 说明：该示例刻意覆盖本技能的所有规则点（Port/Adapter、依赖方向、薄入口厚用例、构造注入 clock/idGen、错误边界、事务 port、可测试性、DTO/Command/Result、命名）。

### 1) 业务侧：Domain + Ports + Errors + Use Case（不依赖 gin/gorm/SDK）

```go name=internal/order/order.go
package order

import (
	"context"
	"errors"
	"time"
)

var (
	// 错误边界：对外暴露业务错误，不泄漏外部错误细节
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidAmount = errors.New("invalid amount")
	ErrPaymentFailed = errors.New("payment failed")
)

// 边界契约：用例输入/输出使用稳定 DTO（Command/Result）
type CreateOrderCommand struct {
	UserID string
	Amount int64
}

type CreateOrderResult struct {
	ID        string
	UserID    string
	Amount    int64
	CreatedAt time.Time
}

// 领域模型：反映业务语言，而不是 ORM 表结构
type Order struct {
	ID        string
	UserID    string
	Amount    int64
	CreatedAt time.Time
}

// Ports：业务侧抽象
type OrderRepository interface {
	Save(ctx context.Context, o *Order) error
}

type PaymentGateway interface {
	Charge(ctx context.Context, userID string, amount int64) error
}

// 外部性 Ports：时间与 ID
type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }

// 事务 Port：一致性由用例表达，事务由基础设施实现
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

```go name=internal/order/create_order_uc.go
package order

import "context"

// 用例：厚，用来编排业务流程
type CreateOrderUseCase struct {
	tx    TxManager
	repo  OrderRepository
	pay   PaymentGateway
	clock Clock
	idGen IDGenerator
}

// 显式注入：构造函数列出全部依赖
func NewCreateOrderUseCase(tx TxManager, repo OrderRepository, pay PaymentGateway, clock Clock, idGen IDGenerator) *CreateOrderUseCase {
	return &CreateOrderUseCase{tx: tx, repo: repo, pay: pay, clock: clock, idGen: idGen}
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, cmd CreateOrderCommand) (*CreateOrderResult, error) {
	if cmd.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	var out *CreateOrderResult
	if err := uc.tx.WithinTx(ctx, func(txCtx context.Context) error {
		o := &Order{
			ID:        uc.idGen.NewID(),
			UserID:    cmd.UserID,
			Amount:    cmd.Amount,
			CreatedAt: uc.clock.Now(),
		}

		// 编排属于用例：调用顺序、失败策略、幂等/状态机/权限（按需扩展）
		if err := uc.pay.Charge(txCtx, cmd.UserID, cmd.Amount); err != nil {
			return ErrPaymentFailed
		}
		if err := uc.repo.Save(txCtx, o); err != nil {
			return err
		}

		out = &CreateOrderResult{ID: o.ID, UserID: o.UserID, Amount: o.Amount, CreatedAt: o.CreatedAt}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}
```

### 2) 基础设施侧：Adapters（对接外部 + 转换数据/错误/语义）

```go name=internal/infra/persistence/gorm_tx_manager.go
package persistence

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

type GormTxManager struct {
	db *gorm.DB
}

func NewGormTxManager(db *gorm.DB) *GormTxManager {
	return &GormTxManager{db: db}
}

func (m *GormTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func dbFrom(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if v := ctx.Value(txKey{}); v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx
		}
	}
	return fallback
}
```

```go name=internal/infra/persistence/gorm_order_repo_adapter.go
package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"example.com/myapp/internal/order"
)

type orderModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string
	Amount    int64
	CreatedAt time.Time
}

type GormOrderRepoAdapter struct{ db *gorm.DB }

func NewGormOrderRepoAdapter(db *gorm.DB) *GormOrderRepoAdapter { return &GormOrderRepoAdapter{db: db} }

func (r *GormOrderRepoAdapter) Save(ctx context.Context, o *order.Order) error {
	db := dbFrom(ctx, r.db).WithContext(ctx)

	m := orderModel{ID: o.ID, UserID: o.UserID, Amount: o.Amount, CreatedAt: o.CreatedAt}
	if err := db.Create(&m).Error; err != nil {
		// 错误语义转换示例：外部错误 -> 业务错误
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return order.ErrConflict
		}
		return err
	}
	return nil
}
```

```go name=internal/infra/system/clock_adapter.go
package system

import "time"

type SystemClock struct{}

func NewSystemClock() SystemClock     { return SystemClock{} }
func (SystemClock) Now() time.Time    { return time.Now() }
```

```go name=internal/infra/system/id_generator_adapter.go
package system

import "github.com/google/uuid"

type UUIDGenerator struct{}

func NewUUIDGenerator() UUIDGenerator { return UUIDGenerator{} }
func (UUIDGenerator) NewID() string   { return uuid.NewString() }
```

```go name=internal/infra/payment/dummy_gateway_adapter.go
package payment

import (
	"context"

	"example.com/myapp/internal/order"
)

// Rule 2: 对接第三方支付 SDK/HTTP 的地方，必须封装为 adapter
type DummyGatewayAdapter struct{}

func NewDummyGatewayAdapter() *DummyGatewayAdapter { return &DummyGatewayAdapter{} }

func (g *DummyGatewayAdapter) Charge(ctx context.Context, userID string, amount int64) error {
	_ = ctx
	_ = userID
	_ = amount
	return nil
}

var _ order.PaymentGateway = (*DummyGatewayAdapter)(nil)
```

### 3) Web 层：薄 Handler（只做转发 + 映射，不做业务编排）

```go name=internal/httpapi/order_handler.go
package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/myapp/internal/order"
)

// Rule 3: handler 薄：bind/validate -> call usecase -> map response
type OrderHandler struct {
	createUC *order.CreateOrderUseCase
}

func NewOrderHandler(createUC *order.CreateOrderUseCase) *OrderHandler {
	return &OrderHandler{createUC: createUC}
}

type createOrderReq struct {
	UserID string `json:"userId" binding:"required"`
	Amount int64  `json:"amount" binding:"required,gt=0"`
}

func (h *OrderHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/orders", h.create)
}

func (h *OrderHandler) create(c *gin.Context) {
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.createUC.Execute(c.Request.Context(), order.CreateOrderCommand{
		UserID: req.UserID,
		Amount: req.Amount,
	})
	if err != nil {
		// Rule 7: handler 只认识业务错误，不判断 gorm/SDK error
		switch err {
		case order.ErrInvalidAmount:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case order.ErrConflict:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        res.ID,
		"userId":    res.UserID,
		"amount":    res.Amount,
		"createdAt": res.CreatedAt,
	})
}
```

### 4) Composition Root：在 main.go 手写组装（允许跨层 new 的唯一地方）

```go name=cmd/myapp/main.go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"example.com/myapp/internal/httpapi"
	"example.com/myapp/internal/infra/payment"
	"example.com/myapp/internal/infra/persistence"
	"example.com/myapp/internal/infra/system"
	"example.com/myapp/internal/order"
)

func main() {
	// 外部依赖初始化
	db, err := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Adapters（实现 Ports）
	txManager := persistence.NewGormTxManager(db)
	orderRepo := persistence.NewGormOrderRepoAdapter(db)
	payGateway := payment.NewDummyGatewayAdapter()
	clock := system.NewSystemClock()
	idGen := system.NewUUIDGenerator()

	// UseCase（显式注入）
	createOrderUC := order.NewCreateOrderUseCase(txManager, orderRepo, payGateway, clock, idGen)

	// Web（薄 handler）
	r := gin.Default()
	h := httpapi.NewOrderHandler(createOrderUC)
	h.RegisterRoutes(r)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
```

### 5) 可测试性：用 fake/memory adapter + 可控 clock/idGen 单测用例
```go name=internal/order/create_order_uc_test.go
package order_test

import (
	"context"
	"testing"
	"time"

	"example.com/myapp/internal/order"
)

type fakeTx struct{}
func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type memRepo struct{ saved *order.Order }
func (r *memRepo) Save(ctx context.Context, o *order.Order) error { r.saved = o; return nil }

type okPay struct{}
func (okPay) Charge(ctx context.Context, userID string, amount int64) error { return nil }

type fixedClock struct{ t time.Time }
func (c fixedClock) Now() time.Time { return c.t }

type fixedID struct{ id string }
func (g fixedID) NewID() string { return g.id }

func TestCreateOrder_HappyPath(t *testing.T) {
	repo := &memRepo{}
	uc := order.NewCreateOrderUseCase(
		fakeTx{},
		repo,
		okPay{},
		fixedClock{t: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)},
		fixedID{id: "order-123"},
	)

	res, err := uc.Execute(context.Background(), order.CreateOrderCommand{UserID: "u1", Amount: 100})
	if err != nil { t.Fatal(err) }

	if res.ID != "order-123" { t.Fatalf("id=%s", res.ID) }
	if repo.saved == nil { t.Fatalf("expected saved order") }
	if repo.saved.CreatedAt != res.CreatedAt { t.Fatalf("createdAt mismatch") }
}

```

## 从SOLID原则看本技能
> 本技能是SOLID原则的具体可执行的落地方案，没有冲突，反而是对SOLID原则的补充和细化。SOLID原则提供了面向对象设计的指导，而本技能则将这些原则具体化为可执行的步骤和检查点，帮助开发者在实际编码过程中贯彻这些设计原则，从而实现更清晰、更可维护的架构设计。

### S — SRP（单一职责原则）
**含义**：一个类或模块应该有且只有一个引起它变化的原因。即：一个类只负责一件事。
- 入口层（handler/consumer/cli）只负责协议与 IO 映射
- 用例层负责业务流程编排
- adapter 负责外部交互与转换（数据/错误/语义）

**反例**：handler 里开事务、查库、调 SDK、做状态机。

### O — OCP（开闭原则）
**含义**：对扩展开放，对修改关闭。即：新增功能时，应尽量通过扩展现有代码实现，而不是修改已有的核心代码。
- 扩展通过新增/替换 adapter 或新增用例实现，而不是修改业务核心到处插 if/else
- 用例依赖抽象 port；新增外部实现不改用例

**反例**：在用例里写 `switch provider { case A: ... case B: ... }` 直接调用不同 SDK。

### L — LSP（里氏替换原则）
**含义**：子类对象必须能够替换掉父类对象，且程序行为不变。即：继承要确保子类不破坏父类的功能契约。
- 任何 adapter 实现 port 时，必须满足业务对该 port 的语义约定
    - 例如：`FindByID` 找不到返回 `ErrNotFound`（或返回 nil+nil），所有实现都一致
    - 不要某实现返回 `nil,nil`，另一个实现返回 `ErrNotFound`，导致用例行为不稳定

**反例**：同一个 port 在不同实现里“成功/失败语义”不一致，导致用例在替换实现后逻辑崩坏。

### I — ISP（接口隔离原则）
**含义**：不应强迫客户端依赖它不需要的接口。即：将臃肿的大接口拆分为多个专门的小接口。
- port 要“小而清晰”，不要搞一个巨大的 `Repository`/`Service` 包含几十个方法
- 按用例需要拆分接口：`OrderReader`、`OrderWriter`、`PaymentCharger` 等（按团队习惯命名）

**反例**：`type UserRepository interface { Find... Save... Delete... List... Batch... Tx... Cache... }` 变成垃圾桶。

### D — DIP（依赖倒置原则）
**含义**：依赖抽象而非具体实现。即：高层模块不应依赖低层模块，二者都应依赖抽象接口。
- 用例/领域只依赖 port（接口），不依赖 gorm/gin/sdk client
- 依赖通过构造函数注入（避免全局单例、包级变量、service locator）

**反例**：用例中直接 `time.Now()` / `uuid.New()` / `gorm.DB` / `gin.Context`。


## 注意事项

-  本示例代码只是为了简单演示技能点，实际项目中可能需要更复杂的错误处理、更丰富的领域模型、更细化的 Ports/Adapters 划分、更完善的测试覆盖等。技能的核心在于**遵循原则**，而不是代码细节。
