# 指令管道架构（未来计划）

> 本文档描述的是**未来计划中的架构方向**，不是当前工程的实现。
> 当前代码仍使用旧版 Pipeline（`server/pkg/pipeline/`），待重构后以本文档为准。

---

## 核心思想

类比 HTTP 中间件模型：

```
HTTP:   Request(read-only) → Middleware chain → Response(write-only)
你的:   Pipeline(read-only fields) → Middleware chain → Pipeline(result)
```

- Pipeline 是请求上下文，**不可变字段中间件不可修改，可变字段逐步构建**
- 中间件链**永不中断**，所有中间件都执行完毕，错误全量收集
- 审计日志作为最后一个中间件自动执行，不散落在业务代码中

---

## Pipeline 结构体

```go
type MiddlewareError struct {
    Name string // 中间件名称
    Err  string // 错误描述
}

type Pipeline struct {
    // ===== 不可变字段（创建时设置，中间件不可修改）=====
    Type      string    // 数据源具体类型: mysql / redis / es / prometheus
    TypeGroup string    // 数据源类型分组: sql / nosql / search / mq
    Raw       string    // 原始输入文本
    UserID    string    // 操作人（直接执行=JWT用户，工单执行=申请人）

    // ===== 可变字段（中间件可读写）=====
    OpType     OpType            // read / write / dangerous / unknown
    Fields     map[string]any    // 请求传入的动态字段（不可变，创建时预填）
    Extensions map[string]any    // 中间件补充的信息（可追加）
    Errors     []MiddlewareError // 错误收集，不中断链
    Result     any               // 执行结果
}
```

### 三个 map 的分工

| 字段 | 谁填的 | 举例 | 是否可变 |
|---|---|---|---|
| `Fields` | **Bind 时**从 DTO 填入 | SQL: `{database: "mydb"}`，Redis: `{db: 0}`，ES: `{method: "GET", path: "/_search"}` | ❌ 不改 |
| `Extensions` | **中间件**逐步补充 | SQLParser 加 `{tables: ["users"]}`，未来加 `{columns: ["name","phone"]}` | ✅ 可追加 |
| `Errors` | **中间件**追加 | `{name: "SQLParser", err: "语法错误"}` | ✅ 可追加 |

**Fields vs Extensions 的语义区别：**

- `Fields` = 用户告诉我们的（请求参数，Bind 时预填）
- `Extensions` = 我们分析出来的（中间件解析补充）

---

## 中间件

### 签名

```go
type Middleware func(ctx context.Context, p *Pipeline)
```

不返回 error。所有中间件都执行完，错误全收集在 `p.Errors` 里。

### 中间件列表

| 中间件 | 职责 | 读什么 | 写什么 |
|---|---|---|---|
| `SQLParser` | 解析 SQL，分类 OpType，提取表名 | `TypeGroup`, `Raw` | `OpType`, `Extensions["tables"]` |
| `RedisParser` | 解析 Redis 命令，分类 OpType | `Type`, `Raw` | `OpType` |
| `MongoParser` | 解析 Mongo 命令，分类 OpType | `Type`, `Raw` | `OpType` |
| `ESParser` | 解析 ES 请求，分类 OpType | `Type`, `Raw` | `OpType` |
| `MQParser` | 解析 MQ 操作，分类 OpType | `Type`, `Raw` | `OpType` |
| `PrometheusParser` | 解析 PromQL，分类 OpType | `Type`, `Raw` | `OpType` |
| `RuleEngine` | 匹配配置化规则，覆盖 OpType | `TypeGroup`, `Type`, `OpType` | `OpType`（可能覆盖） |
| `PermissionCheck` | 表级/资源级权限校验 | `UserID`, `Extensions["tables"]` | `OpType`（可能改为 dangerous） |
| `SensitiveField` | 敏感字段脱敏/拦截 | `Extensions["columns"]` | `Result`（脱敏后） |
| `AuditLog` | 记录审计日志 | 全部 Pipeline 字段 | 写数据库，不改 Pipeline |

### 自判断模式

每个中间件根据 `p.Type` 或 `p.TypeGroup` 判断是否处理，不处理直接 return：

```go
func SQLParser(ctx context.Context, p *Pipeline) {
    if p.TypeGroup != "sql" {
        return // 不关我事，跳过
    }
    // 解析 SQL，设 OpType，提取表名
}

func RedisParser(ctx context.Context, p *Pipeline) {
    if p.Type != "redis" {
        return // 跳过
    }
    // 解析 Redis 命令
}
```

**加新数据源类型 = 写一个 parser 函数 + 一行 `.use()`。** 已有中间件零改动。

---

## 链式执行

### 执行规则

1. **所有中间件都执行**，链永不中断
2. 中间件通过 `p.Errors` 收集错误，不通过 return error 中断
3. 审计日志**永远在最后**，确保任何情况都有记录
4. Executor 检查 `p.Errors` 和 `p.OpType` 决定是否执行

### 执行流程

```
Pipeline 创建（Bind 时）
  │  Type, TypeGroup, Raw, UserID 已设
  │  Fields 已预填
  ↓
[Middleware Chain]
  ├─ SQLParser      → 设 OpType，补充 Extensions["tables"]
  ├─ RedisParser    → 跳过（TypeGroup != "nosql"）
  ├─ MongoParser    → 跳过
  ├─ RuleEngine     → 匹配规则，可能覆盖 OpType
  ├─ PermissionCheck→ 校验权限，可能覆盖 OpType
  ├─ SensitiveField → 脱敏处理
  ├─ Executor       → 检查 OpType + Errors，决定执行
  └─ AuditLog       → 记录一切
  ↓
返回 Result 或 Errors 给前端
```

### Executor 的逻辑

```go
func Executor(ctx context.Context, p *Pipeline) {
    // 有错误 → 不执行
    if len(p.Errors) > 0 {
        return
    }

    switch p.OpType {
    case OpDangerous:
        return // 不执行，审计日志已记录
    case OpWrite:
        // 走工单（在 Executor 外部处理，Executor 不处理 write）
        return
    case OpUnknown:
        p.Errors = append(p.Errors, MiddlewareError{Name: "Executor", Err: "unrecognizable statement"})
        return
    case OpRead:
        // 执行
        p.Result = doExecute(p)
    }
}
```

---

## 分入口设计

### 为什么分入口

不同数据源类型需要的请求参数不同，无法用一个 DTO 统一：

| 类型 | 请求参数 |
|---|---|
| **SQL** | `sql` + `database`（可选 schema） |
| **Redis** | `command` + `db`（数字） |
| **ES** | `method` + `path` + `body` |
| **Mongo** | `command`（JSON）+ `database` |
| **MQ** | `topic` + `message` + `headers` |
| **Prometheus** | `promql` |

### 路由设计

```
POST /api/v1/execute/sql       → SQLExecuteRequest{database, sql}
POST /api/v1/execute/nosql     → NoSQLExecuteRequest{db, command}
POST /api/v1/execute/search    → SearchExecuteRequest{method, path, body}
POST /api/v1/execute/mq        → MQExecuteRequest{topic, message, headers}
POST /api/v1/execute/promql    → PromQLExecuteRequest{promql}
```

### 创建 Pipeline 示例

```go
// SQL
p := &Pipeline{
    Type:      ds.Type,       // "mysql"
    TypeGroup: ds.TypeGroup,  // "sql"
    Raw:       req.Sql,
    UserID:    userID,
    Fields:    map[string]any{"database": req.Database},
}

// Redis
p := &Pipeline{
    Type:      ds.Type,       // "redis"
    TypeGroup: ds.TypeGroup,  // "nosql"
    Raw:       req.Command,
    UserID:    userID,
    Fields:    map[string]any{"db": req.DB},
}

// ES
p := &Pipeline{
    Type:      ds.Type,       // "es"
    TypeGroup: ds.TypeGroup,  // "search"
    Raw:       fmt.Sprintf("%s %s\n%s", req.Method, req.Path, req.Body),
    UserID:    userID,
    Fields:    map[string]any{"method": req.Method, "path": req.Path},
}
```

**中间件链完全复用，不感知入口差异。**

---

## 错误处理

### 原则

- 中间件不通过 return error 中断链
- 错误收集到 `p.Errors`，链走完
- 审计日志一定执行

### 最终响应

```go
if len(p.Errors) > 0 {
    // 返回 400，附带所有错误信息
    return response.BadRequest(c, formatErrors(p.Errors))
}
if p.OpType == OpDangerous {
    return response.Forbidden(c, "operation rejected")
}
return response.Ok(c, p.Result)
```

### 审计日志记录错误

```go
func AuditLog(ctx context.Context, p *Pipeline) {
    saveAudit(AuditRecord{
        UserID:    p.UserID,
        Type:      p.Type,
        Raw:       p.Raw,
        OpType:    string(p.OpType),
        Errors:    p.Errors,   // 错误也记录
        Result:    p.Result,
    })
}
```

**即使执行失败，审计也有完整记录。**

---

## 工单自动执行

### 场景

用户提交工单 → 审批通过 → 系统定时/立即执行指令。

### 关键设计

工单执行时，`UserID` 不是系统用户，而是**工单申请人**：

```go
ticket, _ := GetTicketByID(ticketID)
p := &Pipeline{
    Type:      ticket.Type,
    TypeGroup: ticket.TypeGroup,
    Raw:       ticket.InstructionJSON,
    UserID:    ticket.ApplicantID,  // ← 申请人，不是系统
}
chain.Run(ctx, p)
```

**中间件链完全复用。** AuditLog 无差别对待，只读 `p.UserID`，不关心来源。

### 定时执行

工单 DTO 支持 `scheduled_at` 字段：

```go
type CreateTicketRequest struct {
    ProjectID       string `json:"project_id"`
    Title           string `json:"title"`
    Description     string `json:"description"`
    InstructionJSON string `json:"instruction_json"`
    ApprovalMode    string `json:"approval_mode"`     // any_one / all
    ScheduledAt     string `json:"scheduled_at"`       // 可选，ISO 时间
}
```

审批通过后：

```
if ticket.ScheduledAt == "" {
    // 立即执行
    ExecuteTicket(ticket)
} else {
    // 创建定时任务
    cronjob(schedule=ticket.ScheduledAt, ExecuteTicket(ticket))
}
```

---

## 审计日志

### 三个审计事件

```
审计记录 1: 用户 A 提交工单（write 操作）→ action: "ticket_create"
审计记录 2: 审批人 B 审批通过 → action: "ticket_approve"
审计记录 3: 系统执行工单（申请人 A 的指令）→ action: "ticket_execute"
```

**三条记录串起来就是完整的审计链路：** 谁提交的、谁审批的、谁执行的（系统）、谁负责的（申请人）。

### 审计记录结构

```go
type AuditRecord struct {
    ID        string
    UserID    string            // 操作人
    Action    string            // direct_execute / ticket_create / ticket_approve / ticket_execute
    Type      string            // mysql / redis / es
    Raw       string            // 原始输入
    OpType    string            // read / write / dangerous / unknown
    Errors    []MiddlewareError // 执行错误
    Result    string            // 执行结果摘要
    CreatedAt time.Time
}
```

---

## 扩展新数据源类型

### 步骤

1. 在 `DatasourceTypes` 映射表加一行
2. 写一个 Parser 中间件（解析命令 + 分类 OpType）
3. 在入口加一行 `.use(MyParser)`
4. 如果连接方式不同，在 dbpool 加连接函数
5. 如果执行方式不同，在 Executor 加 case

### 示例：加 Prometheus

```go
// 1. 映射表
DatasourceTypes["prometheus"] = {Group: GroupSearch, Label: "Prometheus"}

// 2. Parser 中间件
func PrometheusParser(ctx context.Context, p *Pipeline) {
    if p.Type != "prometheus" {
        return
    }
    p.OpType = OpRead // PromQL 全是读
}

// 3. 注册
chain.Use(PrometheusParser)

// 4. Executor 加 case
case "prometheus":
    return executePromQL(ctx, p)
```

**已有代码零改动。**
