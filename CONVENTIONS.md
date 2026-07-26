# CONVENTIONS.md — 系统架构与项目宪法

> 项目发展的"活档案"。这里只放**跨栈的架构决策和系统设计**。
> 技术栈的具体约定见 `server/CONVENTIONS.md`（后端）和 `web/CONVENTIONS.md`（前端）。
>
> **新增约定**：开发过程中发现未记录的技术约定或用户偏好，直接追加到下文对应章节，
> 不需要经过"写不写"的思考。

---

产品定位：  Data Ops Platform（数据运维平台） — 不是又一个数据库管理工具，而是统一的数据操作网关 +
  审批治理层。核心价值是"所有数据操作经过审批、有审计、可追溯"，不是替代数据库自身的权限体系。


## 项目边界

- 个人项目，优先简单直接
- 不使用 DI 容器
- 不默认创建 interface
- 不默认抽 BaseRepo
- 不默认写测试，除非用户明确要求
- 所有接口字段以代码为准，不维护外部 API 契约文件
- 当前版本用 git log 追踪，不维护独立版本文档
- 开发数据库：SQLite（文件 `demo.sqlite.db`）

## 数据库设计规则

| 规则 | 说明 |
|---|---|
| 每表必有字段 | `id`(varchar 32 PK, UUID v7 无短横线) + `created_at` + `updated_at` + `deleted_at` |
| 外键引用 | `{prefix}_id` 形式（如 `role_id`、`user_id`）|
| 禁止 | BaseModel、外键约束、自增依赖 |
| 模型 | 表结构和 Go model 一一对应 |
| 关联维护 | 代码内维护引用关系，不依赖 GORM 关联特性 |
| 依赖管理 | `go get` 安装/卸载，不直接编辑 go.mod（除非必要），禁止编辑 go.sum |
| 表名前缀 | 数据操作业务表使用 `data_` 前缀，用户角色权限等系统表使用 `sys_` 前缀。通过模型 `TableName()` 方法显式指定 |
| ID 规则 | PK 直接使用 UUID v7（无短横线 32 位），不设冗余的 `uuid` 字段。不依赖自增。`BeforeCreate` 钩子自动填充 |

## 核心概念

| 概念 | 说明 |
|---|---|
| **数据源（Datasource）** | 数据库实例连接信息（类型 `type` + 类型分组 `type_group`、host、port、账号密码、连接池）|
| **项目（Project）** | 工作空间，绑定一个数据源，通过 `scope` 限定资源范围（如数据库列表、db number、index 模式等），范围语义由数据源类型决定 |
| **指令（Instruction）** | 统一的操作指令结构体，与数据源类型无关 |
| **规则引擎（Rule Engine）** | 基于指令分类结果做决策（放行/工单/拦截），不感知数据源类型 |
| **权限检查点（PermissionCheck）** | 扩展点，`(ctx, userID, Instruction) → (bool, error)`，默认无实现即为透传 |
| **工单（Ticket）** | 写操作的审批单元，包含不可变指令快照 |
| **提权（Escalation）** | developer 临时绕过工单的写权限，有 TTL |

## 架构红线（不可违反）

### 1. 双库隔离

```
平台元数据库（GORM）             被管理数据库（原生 database/sql）
┌────────────────────┐          ┌────────────────────────────┐
│ sys_user           │          │ MySQL / PostgreSQL 实例     │
│ data_datasource    │          │                            │
│ data_project       │          │ 执行用户 SQL、浏览库表      │
│ data_ticket        │          │                            │
│ data_audit_log     │          │ 两类连接完全隔离            │
│ ...                │          └────────────────────────────┘
│ ⚠ 不做用户 SQL     │
└────────────────────┘
```

- **GORM 只操作平台元数据库，永不执行用户 SQL**
- 被管理数据库使用原生 `database/sql` 直接执行用户语句
- 两类数据库的驱动、连接池、事务边界完全独立

### 2. 连接池策略

- **懒加载**：系统启动不为数据源建连接池，按 `datasource_id` 第一次 SQL 执行时创建
- **复用**：多项目引用同一数据源 → 复用同一连接池
- **释放**：空闲超时自动释放；配置变更关闭旧池
- **测试隔离**：连接测试用临时连接，用完即关，不进入连接池

### 3. 数据源类型系统

数据源类型分为两层：**具体类型（type）** 和 **类型分组（type_group）**，两者都持久化到 `datasources` 表。

```
Datasource {
    type:       "mysql"        // 具体类型，如 mysql / postgresql / redis / mongo
    type_group: "sql"          // 所属分组，如 sql / nosql / search / mq
}
```

#### 内置类型映射

```go
DatasourceTypes = {
    "mysql":       { Group: "sql",    Label: "MySQL" },
    "postgresql":  { Group: "sql",    Label: "PostgreSQL" },
    "tidb":        { Group: "sql",    Label: "TiDB" },
    "oceanbase":   { Group: "sql",    Label: "OceanBase" },
    "redis":       { Group: "nosql",  Label: "Redis" },
    "mongo":       { Group: "nosql",  Label: "MongoDB" },
    "es":          { Group: "search", Label: "Elasticsearch" },
    "kafka":       { Group: "mq",     Label: "Kafka" },
}
```

- 新增数据源类型 = 修改代码中的映射表 + 实现 Parser/Executor
- 加新类型时 type_group 自动匹配（如加 questdb → type_group = "sql"）
- 该映射表同时作为前端 `/api/v1/datasource-types` 的数据来源

#### 前端数据来源

提供 `/api/v1/datasource-types` 接口，返回按 group 分组的数据源类型列表，前端渲染为分组下拉或复选框，不硬编码任何字符串。

```json
// GET /api/v1/datasource-types
{
  "groups": [
    {
      "group": "sql",
      "label": "SQL 数据库",
      "types": [
        { "type": "mysql", "label": "MySQL" },
        { "type": "postgresql", "label": "PostgreSQL" },
        { "type": "tidb", "label": "TiDB" }
      ]
    },
    {
      "group": "nosql",
      "label": "NoSQL",
      "types": [
        { "type": "redis", "label": "Redis" },
        { "type": "mongo", "label": "MongoDB" }
      ]
    }
  ]
}
```

#### 规则匹配（type_group + type_scope）

规则使用 `type_group`（大类）+ `type_scope`（小类）双字段，决定该规则对哪些数据源生效：

| type_group | type_scope | 匹配效果 |
|-----------|-----------|---------|
| `""` | `""` | 所有数据源 |
| `"sql"` | `""` | 所有 SQL 数据源（mysql/pg/tidb/oceanbase/mariadb） |
| `""` | `"mysql"` | 仅 MySQL |
| `"sql"` | `"mysql"` | 仅 MySQL（双重限定） |

匹配逻辑：type_group 匹配数据源的分组，type_scope 匹配具体类型。空字符串为"全部"：

```go
// WHERE (type_group = ? OR type_group = '') AND (type_scope = ? OR type_scope = '')
```

> 例如：SELECT 规则 `type_group="sql", type_scope=""` → 自动覆盖 mysql/pg/tidb/mariadb
> DROP 规则 `type_group="sql", type_scope=""` → 同样自动覆盖所有 SQL 类型
> FLUSHALL 规则 `type_group="", type_scope="redis"` → 仅匹配 Redis
> 使用频率大多为分组级，"具体类型"仅用于"仅 MySQL/Redis"等精确匹配场景

针对 SQL / Redis / MongoDB / ES 等**所有数据源类型**统一适用：

```
用户输入原始命令
      ↓
  [Parser] 解析为统一指令结构体 Instruction
      ↓
  [Rule Engine] 基于指令做分类决策
      ↓
  [PermissionCheck] 可插拔扩展点（默认透传）
      ↓
  ├─ DANGEROUS → 拒绝执行 + 审计
  ├─ WRITE    → 创建工单 → 审批 → 系统自动执行指令快照
  ├─ READ     → [Executor] 直接执行 → 审计
  └─ UNKNOWN  → 返回错误，不执行
```

- 多语句按类型规则拆分（SQL 按分号，Redis 按换行），逐条匹配
- 任一语句 DANGEROUS → 整批拒绝
- 任一语句 WRITE → 整批走工单
- 查询默认最多 10,000 行，超时 30 秒

#### Instruction 结构体

```go
type DataSourceType string

const (
    DsSQL   DataSourceType = "sql"
    DsRedis DataSourceType = "redis"
    DsMongo DataSourceType = "mongo"
    DsES    DataSourceType = "es"
    DsKafka DataSourceType = "kafka"
)

type OpType string

const (
    OpRead      OpType = "read"      // 直接执行
    OpWrite     OpType = "write"     // 走工单
    OpDangerous OpType = "dangerous"  // 直接拒绝
    OpUnknown   OpType = "unknown"   // 无法识别，返回错误
)

type Instruction struct {
    Type    DataSourceType  // sql / redis / mongo / es / kafka
    Command string          // SELECT / KEYS / get / find
    Args    []string        // ["users"] — 操作参数，由 Parser 提取
    Raw     string          // 原始输入文本
    OpType  OpType          // read / write / dangerous / unknown — Parser 初步判断，Rule Engine 可覆盖
}
```

> 不使用两个 bool（IsRead / IsDangerous），用单个 OpType 表达互斥状态，消除非法组合。
> 不额外添加 "Resources" 等预留字段。PermissionCheck 如需解析资源，
> 可直接使用 Raw + Args 自行提取，框架不做假设。

#### PermissionCheck 扩展点

```go
// 框架定义接口，用户自行实现。默认实现直接返回 true（透传）。
type PermissionChecker interface {
    Check(ctx context.Context, userID string, inst Instruction) (bool, error)
}
```

- 框架只定义签名，不规定管道中的执行顺序（先权限后规则或先规则后权限均可）
- 框架不往 Instruction 中增加任何"预留字段"，保持结构体精简
- 用户实现中可以访问 Instruction 全量信息（Type / Command / Args / Raw），自行解析资源

### 4. 工单状态机

```
pending → approved → executing → executed
   │         │           │
   │         │           └→ execute_failed
   │         │
   │         └→ rejected（任一审批人拒绝）
   │
   └→ rejected（主动撤销 / all 模式下拒绝）
```

- 工单 SQL 在**提交时存快照**，审批后不可修改
- 审批模式：`any_one`（任一通过即执行）/ `all`（全部通过）
- project_admin 可兜底放行，走同一审计
- 通过后系统自动执行快照，不依赖人工操作

### 5. 项目成员角色

在 `data_project_member` 表中通过 role 字段定义用户在项目内的权限：

| 角色 | 权限 |
|------|------|
| `viewer` | 只读查看项目内容、审计日志 |
| `developer` | 可执行 SQL（读）、提工单（写）、申请提权 |
| `approver` | 可审批工单、审批提权 |
| `admin` | 项目管理员，拥有以上全部权限 + 管理成员 + 编辑项目配置 |

一个人可以有多个角色（表中多条记录）。审批人员不是从 `data_project` 的独立字段维护，而是通过 member 表中的 `approver` 角色统一管理。

### 6. 权限模型

权限码三段式设计：`模块:功能:动作`（如 `db:sql:read`）。权限判断基于**段级通配匹配**：

#### 数据表（5 张）

| 表 | 说明 |
|---|---|
| `sys_user` | 纯用户信息，无 role_code。必有 id + timestamps |
| `sys_role` | 角色定义（如 sql_dev、platform_admin）|
| `sys_permission` | 权限码（如 `db:project:access`、`*`）|
| `sys_user_role` | 用户-角色关联。通过 `user_id` + `role_id` 关联 |
| `sys_role_permission` | 角色-权限码关联 |

#### 匹配规则

```
*             → 超级管理员，匹配全部
db:*          → 匹配所有 db:xxx:xxx（等价 db:*:*）
db:datasource:* → 匹配 db:datasource 下所有操作
db:*:view     → 匹配 db 模块下所有资源的 view 操作
db:project:view → 精确匹配
*:view        → 禁止（* 不在段首）
```

权限码和 API 接口是 **1 对多**关系。一个 `db:project:access` 覆盖多个 GET 接口。

**业务特判在 handler 内做**（如 owner/提权判断），中间件只检查入口权限码。

> `*` 权限码默认赋予 `platform_admin` 角色。

### 7. 核心数据流（完整链路）

```
用户登录 → JWT Token
   ↓
进入项目 → 确定身份（项目内角色）→ 选择数据源
   ↓
输入原始命令 / 选择已有指令
   ↓
[Parser] 按数据源类型解析为 Instruction
  │  Instruction {
  │    type:    "sql"          (sql/redis/mongo/es/...)
  │    command: "SELECT"
  │    args:    ["users"]
  │    raw:     "SELECT * FROM users"
  │    op_type: "read"         (read / write / dangerous / unknown)
  │  }
  ↓
[Rule Engine] 匹配规则表
  │  规则不限类型，按 priority 排序，第一条命中即生效
  │  可覆盖 op_type（如 Parser 判为 write，规则命中 dangerous 则改为 dangerous）
  ↓
[PermissionCheck] 扩展点（默认透传）
  │  (ctx, userID, Instruction) → (true, nil)
  ↓
  ├── dangerous   → 拒绝 + 审计日志
  ├── write       → 创建工单 → 审批 → 系统自动执行快照 → 审计 + Webhook
  ├── read        → [Executor] 执行 → VTable/JSON 展示 → 审计
  └── unknown     → 返回错误，不执行
```

### 8. 关键安全默认

- DROP/TRUNCATE/ALTER 默认禁止
- 密码读写走统一入口（`pkg/password`），为后续加密留坑位
- 审计只存元信息（操作人、指令、耗时、状态、IP），不存结果集
- 多语句中任意语句不可识别 → 返回错误，不降级执行
- 响应格式统一：`{ code, msg, data }`

## 当前不做

- SSO / OAuth / LDAP
- 字段级 / 行级权限（数据库自身权限体系应做好）
- 查询结果脱敏与导出
- 审批组 / 顺序审批
- Webhook 回调审批
- 可视化增删改查
- 国际化（代码保留但不再迭代，未来按需启用）
- Shell / SSH / 堡垒机（不在产品边界内）

## 决策记录

> 按时间倒序排列。每次出现新的技术偏好、弃用、重构决策，追加到最前面。

### 2026-06-30 Driver 包重构：SQLConnector + 按 type_group 分发

**背景：** 加一个新数据源类型需要改 10+ 处 switch（dbpool、test、meta、exec、ticket），且 buildDSN 在三处重复。列类型信息只返回列名，不带 VARCHAR 长度/NOT NULL 等元数据。全项目 8 处 switch 靠人工同步，极易漏改。

**决策：**

| 决策 | 说明 |
|------|------|
| **pkg/driver 包引入 SQLConnector 接口** | SQL 族（MySQL/PG/TiDB/OceanBase/MariaDB）共用一个 typed 接口，提供 DSN/SetDatabase/ListDatabases/ListTables/ListColumns。返回 *sql.DB，不存在 any 类型泄漏 |
| **非 SQL 类型不强行统一** | Redis/MongoDB/ES 各自独立连接函数（无通用接口，避免 any 泄漏），不继承 SQLConnector。按 type_group 在 exec/ticket 层面分发 |
| **ScanRows 通用函数** | 使用 `rows.ColumnTypes()` 读取列名称、数据库类型名、长度、精度、Nullability。`QueryResult.Columns` 从 `[]string` 改为 `[]ColumnInfo` |
| **dbpool 使用 driver 注册表** | 删除 3 份 buildDSN 重复代码，统一走 `GetSQLConnector(ds.Type).DSN()`。SQL/Redis/Mongo/ES 四种连接池类型各自独立管理 |
| **按 TypeGroup 分发执行** | `exec.Execute()` 和 `ticket.ExecuteTicket()` 改为按 `ds.TypeGroup`（sql/nosql/search）分发，内部再按 type 选择具体实现 |

**扩展方式：**
- 加新 SQL 类型：`mysql.go` 里 `init()` 注册一行 `RegisterSQLConnector("sqlserver", &mssqlConnector{})`，零处改已有代码
- 加新 NoSQL 类型：新建 `pkg/driver/xxxx.go`，实现连接函数，在 exec 的 nosql 分支加 type case
- 列类型信息：SQL 族自动通过 `rows.ColumnTypes()` 获取，无需 SQL 方言改造

**Why：** 不搞万能抽象接口（避免 any 强转风险），按 type_group 隔离接口边界。switch 集中到 2 个调度点（exec/ticket），每个点只按 group 分发，group 内部按 type 分发，两层总数不超过 4 个 switch，不会因加新类型而无限膨胀。

### 2026-06-30 数据模型重构：表名前缀 + 统一 ID 规则 + 项目模型优化

**背景：** 数据表命名不一致（`db_project` vs `datasources` vs `users`），ID 规则约定有冗余（uint PK + uuid 列），项目模型有 SQL 中心主义设计缺陷（`databases` 字段不适用于非 SQL 数据源），审批人管理采用逗号分隔字符串的设计瑕疵。

**决策：**

| 决策 | 说明 |
|------|------|
| **双前缀体系 `data_` / `sys_`** | 数据操作业务表使用 `data_` 前缀（`data_project`、`data_datasource`、`data_ticket` 等），用户角色权限等系统表使用 `sys_` 前缀（`sys_user`、`sys_role`、`sys_permission` 等）。通过 `TableName()` 显式指定 |
| **ID 规则简化** | PK 直接使用id 类型 UUID v7（32 位无短横线），废除 uint PK + uuid 冗余列。`BeforeCreate` 钩子自动填充 |
| **databases → scope** | 项目绑定数据源后不再使用 `databases`（SQL 术语），改为 `scope`（JSON 文本）。由前端根据数据源类型渲染不同的输入控件（SQL→库名列表，Redis→db number，ES→index pattern）。不额外加 scope_type 字段，因为通过 datasource.type 可推导 |
| **approver_ids 移除 → 成员角色合并** | 废弃 `db_project.approver_ids`（逗号分隔字符串）和 `auto_match_approver`。审批人角色 `approver` 直接作为 `data_project_member` 的一种角色，与 viewer / developer / admin 统一管理 |
| **project_owner → admin** | 成员角色中 `project_owner` 改为 `admin`（项目管理员），语义更准确，避免"唯一归属"的暗示。同时新增 `approver` 审批人角色 |
| **ProjectMember 补全时间戳** | `data_project_member` 增加 `updated_at`、`deleted_at` 字段，遵守每表必有四条时间戳的约定 |

### 2026-06-30 设计原则确立：「设计正确返工少」

**背景：** 用户表达了对设计质量的核心要求——不在乎改动量大小，只在乎一次做对。

**决策：**

| 决策 | 说明 |
|------|------|
| **设计正确优先于改动量** | 宁可按正确方式全部重来，也不要妥协留下设计缺陷。不因为"改动大"而选择"差不多"的方案 |
| **发现问题立即修正** | 项目早期发现的设计问题直接改，不积攒技术债。SQLite 开发库没有数据迁移成本，是修正设计的最佳窗口 |
| **记录在设计文档中** | 每次设计决策必须记录到 `CONVENTIONS.md` 的决策记录，形成可追溯的设计演进历史 |

**Why：** 项目处于早期阶段，元数据库（SQLite）无数据迁移成本，前端路由级组件尚未深度耦合。此时修正设计缺陷的代价是最低的，越往后拖改造成本越高。

**背景：** 规则（`datasource_rule`）之前使用 `db_type` 字段做精确匹配，导致同一条规则（如 SELECT）需要为 mysql、pg 各写一条，也缺乏对"SQL 类数据源"的天然分组支持。

**决策：**

| 决策 | 说明 |
|------|------|
| **Datasource 分层：type + type_group** | `type` 为具体类型（mysql/redis），`type_group` 为所属分组（sql/nosql/search），两者都持久化到数据库 |
| **Rule.type_group + type_scope 双字段** | `type_group` 匹配大类（sql/nosql/search），`type_scope` 匹配小类（mysql/redis/es）。两者都留空 = 全部，type_group 匹配分组，type_scope 匹配具体类型 |
| **前端 API 驱动** | `GET /api/v1/datasource-types` 返回分组数据结构，前端渲染不硬编码 |
| **加新类型只需改映射表** | 在映射表加一行即可，已有 `type_group="sql"` 的规则自动覆盖 |

**匹配逻辑：** `(type_group = dsGroup OR type_group = '') AND (type_scope = dsType OR type_scope = '')`

### 2026-07-05 统一动态筛选协议 + 通用分页 Handler

**背景：** 所有列表接口的 handler 和 repo 层重复 5+ 行样板代码（NeedCount 默认值、Bind、Normalize），且前端 DataTable 已定义筛选操作符约定（`=value`、`=～*val*` 等），但后端完全硬编码每个 filter 字段。

**决策：**

| 决策 | 说明 |
|------|------|
| **新增 `pkg/queryfilter/`** | 通用动态筛选器：`Parse`（解析约定前缀）→ `ApplyToDB`/`ApplyAll`（安全注入 GORM）→ `Paginate`（通用分页） |
| **新增 `response.ParseListQuery()`** | handler 层统一入口，替代 `q.NeedCount=true` + `Bind` + `Normalize` 三行样板 |
| **Repo 签名切换** | 从 `List(ctx, query ListQuery)` 改为 `List(ctx, pq PageQuery, filters map[string]string)` |
| **防注入设计** | `ApplyAll` 通过 `columnMap`（参数名 → 列名）限制可用列，不认识的参数静默忽略，值永远走 GORM `?` 参数化绑定 |
| **删除 8 个 `ListQuery` 结构体** | user/datasource/audit/project/dsrule/webhook/escalation/ticket 的 DTO 不再需要 |

**效果：** handler 从 12 行缩到 4 行，repo filter 瀑布变成 `ApplyAll` + `Paginate` 两行。前端可传 `status==active`、`name__contains==～*admin*` 等任意约定前缀值。

### 2026-06-30 指令管道架构 —— 统一 Parser / Rule Engine / Executor / PermissionCheck 模型

**背景：** 当前 SQL 执行流程中，内置硬编码分类器（`sqlclassifier`）和可配置规则表（`dsrule`）两套并行，行为不透明，且整个模型绑定在 SQL 上。未来需要支持 Redis / MongoDB / ES / Kafka 等多种数据源类型。

**决策：**

| 决策 | 说明 |
|------|------|
| **引入 Instruction 统一指令模型** | 所有数据源类型的操作统一为 `{type, command, args, raw, op_type}` 结构体。`op_type` 为单字段枚举（read/write/dangerous/unknown），替代两个 bool，消除非法组合。不添加任何"预留字段" |
| **Parser 按类型可插拔** | 每种数据源一个 Parser，输入原始文本，输出 Instruction。sql parser 用正则（不深解析 AST），redis/mongo 等各写专属 parser |
| **Rule Engine 不感知类型** | 规则表配置 pattern + action，按 priority 排序，第一条命中即生效。与数据源类型无关 |
| **PermissionCheck 为扩展点** | 框架只定义 `(ctx, userID, Instruction) → (bool, error)` 签名，不规定实现也不规定管道顺序。默认透传 |
| **Executor 按类型可插拔** | 每种数据源一个 Executor，接收 Instruction，执行并返回结果 |
| **项目支持多数据源绑定** | 一个项目绑定多个数据源，每个绑定细化到资源范围（如 MySQL db1,db2 / Redis db0-5 / Mongo collection_*）|
| **不引入库表级精细权限** | 规则引擎定位是操作分类（read/write/dangerous），不是数据库权限替代品。表级权限由数据库自身 GRANT 体系负责。PermissionCheck 扩展点为需要此能力的用户预留 |

**数据流：** 用户输入 → Parser → Instruction → Rule Engine → [PermissionCheck] → 决策（放行/工单/拦截）→ Executor → 审计

**扩展方式：**
- 新增一种数据源类型 = 新增 Parser 实现 + Executor 实现 + 数据源类型枚举。已有代码零改动
- 新增资源级权限校验 = 实现 PermissionCheck 接口插入管道。框架零改动

**Why：** 规则引擎统一决策意味着审批流、审计日志、Webhook 通知完全复用，不需要为每种类型重复造轮子。正则过滤不深解析 AST，跨方言兼容性好，不会因为不认识的语法崩溃。PermissionCheck 保持极简签名，不给 Instruction 加多余字段，确保接口稳定。

### 2026-06-28 权限模型重构：role_code → 权限码三段式
- 废弃双层角色模型（role_code + 业务表），改用权限码三段式（`模块:功能:动作`）+ 前缀 `*` 通配匹配
- 新增 `permissions`、`roles`、`user_roles`、`role_permissions` 4 张表
- 移除 `users.role_code` 字段，用户表只有纯用户信息
- 权限码和 API 接口为 1 对多关系
- 新增数据库设计规则：每表必有 id/uuid/timestamps，禁止 BaseModel/外键/自增
- UUID v7 无短横线，统一使用 `pkg/uuid.NewV7()`

### 2026-06-28 文档重构定稿
- 根 `CLAUDE.md` = `AGENTS.md` 完全一致（copy 关系）
- 根 `CONVENTIONS.md` 仅保留系统架构 + 项目宪法，不包含技术栈约定
- 技术栈约定全部下沉到 `server/CONVENTIONS.md` 和 `web/CONVENTIONS.md`
- 移除国际化（i18n），代码保留但不再迭代
- 移除 `沟通约定` 章节（已在 AGENTS.md 中）
- AGENTS.md 新增"知识沉淀"和"自动优化"规则

### 2026-06-28 统一文档结构：CONVENTIONS.md 引入
- 概念分离：AGENTS = AI 行为规则 + 项目入口，CONVENTIONS = 项目技术约定 + 决策记录
- 删除 `docs/` 下冗余文件（archive、product、requirements、version、progress.yaml）
- 删除 `web/docs/dev-paradigm.md`、`web/README.md`
- 删除 `server/pkg/cache/README.md`、`server/pkg/database/README.md`
- 删除 AGENTS.md（旧版）、README.md、ARCHITECTURE.md

### 2026-06-28 dev.sh 重写
- 改用纯端口杀进程（PID 文件方式废弃）
- 端口固定：server=8080, web=5173
- 命令：start/stop/restart/web/server/status/logs

### 2026-06-07 项目初始化
- 前端：React 19 + TypeScript + Vite + pnpm + shadcn/ui + TailwindCSS 4
- 后端：Go + Echo + GORM + SQLite（开发）
- 路由：react-router-dom v7 createBrowserRouter
- 状态管理：@tanstack/react-query（服务端）+ zustand（客户端）
- HTTP 客户端：ofetch
- 表格：@tanstack/react-table + shadcn/ui Table
- 图标：lucide-react + @tabler/icons-react
- 开发数据库：SQLite 文件 demo.sqlite.db
- 被管理数据库：MySQL / PostgreSQL
- 元数据库和被管理库严格隔离
- 关闭版本文档追踪，用 git log 替代
- 删除 api_contract.yaml，接口以代码为准
