# server/CONVENTIONS.md — 后端技术约定

> 修改 server/ 代码前必读。
> 系统架构与项目级约定见 ../CONVENTIONS.md。

## 项目固定规则

- Module 固定为 `czwlinux.cloud/go-friday-starter`
- 不主动修改 go.mod 中的 Go 版本
- 使用 Echo，不替换为 Gin
- 使用 GORM
- 个人项目优先简单直接
- 不引入 DI 容器
- 不默认创建 interface
- 不默认抽 BaseRepo
- 不默认写测试，除非用户明确要求

## 快速概览

| 类别 | 约定 |
|---|---|
| 框架 | Go + Echo（不替换为 Gin）|
| ORM | GORM（仅限平台元数据库）|
| Module | `czwlinux.cloud/go-friday-starter`（不修改 Go 版本）|
| 全局变量 | `global.Cfg` / `global.Log` / `global.DB` / `global.Engine` / `global.JWT` |
| Echo 实例名 | `global.Engine`（不用 `global.E`）|
| API 响应 | 统一 `{ code, msg, data }` |
| JSON 字段 | 默认 snake_case |
| 密码字段 | 必须 `json:"-"` |
| 权限判断 | 统一 `features/auth/authorization.go`，不散落 |
| 权限匹配 | 前缀 `*` 通配（详见 `docs/permission.md`）|
| UUID 生成 | `pkg/uuid.NewV7()`，返回 32 位无短横线 |
| 密码读写 | 统一 `pkg/password` 入口 |
| 业务代码 | `features/` 下按模块分包 |
| 可复用能力 | `pkg/` 下，不放业务模块 |
| 禁止目录 | utils / common / infra |
| 验证 | 修改 Go 代码后 `gofmt` + `go build ./...` |

## 代码结构规则

- 共享基础设施使用 `global.Cfg`、`global.Log`、`global.DB`、`global.Engine`、`global.JWT`
- Echo 实例名使用 `global.Engine`，不使用 `global.E`
- 业务代码放 `features/name`
- 可复用技术能力放 `pkg/name`
- pkg 不放业务模块
- 不创建 utils、common、infra 这类大筐目录

## 目录职责

| 目录 | 职责 |
|------|------|
| `cmd` | 命令入口和服务启动逻辑 |
| `config` | 配置结构、默认值和 Viper 加载逻辑 |
| `global` | 项目级基础设施组装和全局变量 |
| `routes` | 全局路由注册 |
| `features` | 业务垂直切片（按模块分包） |
| `pkg` | 可复用技术能力，不放业务模块 |
| `migrations` | GORM AutoMigrate |
| `seeds` | 种子数据执行逻辑和 SQL 文件 |

## 模块地图

```
server/
├── cmd/             # 启动入口
├── config/          # 配置加载（viper）
├── global/          # 全局变量（DB、Log、JWT、Engine）
├── pkg/
│   ├── authctx/     # 鉴权上下文
│   ├── cache/       # 缓存（内存/Redis）
│   ├── dbpool/      # 连接池管理（懒加载，SQL/Redis/Mongo/ES）
│   ├── driver/      # 数据源连接器（SQLConnector + 辅助函数）
│   ├── 
│   ├── jwt/         # JWT
│   ├── password/    # 密码统一加解密入口
│   ├── queryfilter/ # 通用动态筛选器（ParseListQuery + ApplyAll + Paginate）
│   └── ...
├── features/
│   ├── auth/        # 认证与授权
│   ├── user/        # 用户管理
│   ├── datasource/  # 数据源管理
│   ├── project/     # 数据项目管理
│   ├── exec/     # 执行（SQL/NoSQL/Search 三路分发）
│   ├── ticket/      # 工单审批流
│   ├── escalation/  # 提权
│   ├── webhook/     # Webhook 通知
│   ├── audit/       # 审计
│   └── ...
├── routes/          # 路由注册
├── migrations/      # 自动迁移
└── seeds/           # 种子数据（required/ 自动执行，test/ 手动）
```

## 初始化流程

```
main.go → cmd.Execute → global.MustInit → global.Engine.Start
```

初始化顺序：config → logger → database → jwt → http

请求流程：route → handler → service → repo → global.DB → response

## 新增业务模块步骤

1. 参考现有 features 下的模块结构
2. 常见文件：model.go、dto.go、repo.go、service.go、handler.go、router.go
3. 简单模块可省略 repo.go，直接在 service 中用 global.DB
4. 新增路由后在 routes 中注册
5. 新增模型后在 migrations/auto.go 注册 AutoMigrate
6. 新增必备数据放 seeds/sql/required（可自动执行）
7. 新增测试数据放 seeds/sql/test（不默认注入）

## 写法规则

- Handler 可以依赖 Echo
- Service 和 Repo 不接收 echo.Context
- Service 和 Repo 第一个参数使用 context.Context
- Repo 查询使用 `global.DB.WithContext(ctx)`
- API 响应固定 `{ code, msg, data }`
- JSON 字段默认 snake_case
- 密码字段必须 `json:"-"`
- 权限判断统一走 `features/auth/authorization.go`，不散落到 handler/service

### 分页列表 Handler 通用化

所有列表 handler 统一使用 `response.ParseListQuery(c)` 替代手写分页样板代码：

```go
func (h *Handler) List(c echo.Context) error {
    pq, filters, err := response.ParseListQuery(c)
    if err != nil {
        return response.BadRequest(c, "invalid param")
    }
    items, total, err := ListXxx(c.Request().Context(), pq, filters)
    if err != nil {
        return response.InternalError(c, "internal error")
    }
    return response.SuccessPage(c, items, total, pq.Page, pq.PageSize)
}
```

### Repo 层通用筛选

repo 层列表函数接收 `pq response.PageQuery` + `filters map[string]string`，使用 `queryfilter` 包处理筛选和分页：

```go
import "czwlinux.cloud/go-friday-starter/pkg/queryfilter"

func List(ctx context.Context, pq response.PageQuery, filters map[string]string) ([]Model, int64, error) {
    var items []Model
    db := global.DB.WithContext(ctx).Model(&Model{})

    // 特殊逻辑（keyword 多列搜索、scope 等）在此处理

    // 通用 filter 自动注入（columnMap 限制可用列，防注入）
    db = queryfilter.ApplyAll(db, filters, map[string]string{
        "status":     "status",
        "project_id": "project_id",
    })

    // 通用分页
    total, err := queryfilter.Paginate(db.Order("id desc"), pq.Page, pq.PageSize, pq.NeedCount, &items)
    return items, total, err
}
```

筛选值遵循 docs/api/README.md 中定义的操作符前缀协议，`queryfilter.Parse` 自动解析：
- `=value` / 无前缀 → 精确匹配
- `=～*val*` → LIKE
- `=[a,b]` → IN
- `=gte:val` / `=lte:val` / `=gt:val` / `=lt:val` → 比较
- `=between:a,b` → BETWEEN

## 数据库设计规则

| 规则 | 说明 |
|---|---|
| 每表必有字段 | `id`(varchar 32 PK, UUID v7 无短横线) + `created_at` + `updated_at` + `deleted_at` |
| 外键引用 | `{prefix}_id` 形式（如 `role_id`、`user_id`）|
| 禁止 | BaseModel、外键约束、自增依赖 |
| 模型 | 表结构和 Go model 一一对应 |
| 关联维护 | 代码内维护引用关系，不依赖 GORM 关联特性 |
| UUID 生成 | 统一使用 `pkg/uuid.NewV7()` |
| 表名前缀 | `data_`（业务表）或 `sys_`（系统表），通过 `TableName()` 显式指定。参见根 CONVENTIONS.md「数据库设计规则」|
| ID 规则 | PK 直接使用 UUID v7（无短横线 32 位），不设冗余 `uuid` 字段。`BeforeCreate` 钩子自动填充 |

## 配置和数据规则

- `config.yml` 可以作为本地真实运行配置
- `config.example.yaml` 是安全示例配置，注释放字段上方
- 新增配置项必须同步 config 结构、加载逻辑、示例配置
- 必备 seed 放 seeds/sql/required，可以自动执行
- 测试 seed 放 seeds/sql/test，不默认注入

## 数据库规则

- 平台元数据库使用 GORM
- 被管理数据库执行用户 SQL 必须使用原生 driver / database/sql，不得使用 GORM
- 两类数据库连接完全隔离

## 提交和验证规则

- 修改 Go 代码后运行 `gofmt`
- 修改代码后优先运行 `go build ./...`
