# 后端开发指引

> **本文档职责**：面向开发者或 AI Agent 的实操指南，解决"怎么改、怎么扩展"的问题。
>
> 不描述系统架构（见 `CONVENTIONS.md`），不描述接口字段（见 `docs/api/`）。
>
> 前端开发指引见 `docs/frontend-development.md`（如有）。

---

## 目录结构

```
server/
├── main.go                  # 入口
├── cmd/                     # cobra 命令（start/version）
├── config/                  # 配置结构体
├── global/                  # 全局变量（DB、JWT、Log）
├── migrations/              # GORM auto-migrate（自动建表）
├── routes/                  # 统一路由注册
├── seeds/                   # 种子数据（SQL 文件，按编号执行）
├── features/                # 业务模块，每个独立包
│   ├── datasource/          # 数据源管理（CRUD + 连接测试）
│   ├── dsrule/              # 规则引擎（正则匹配 + pipeline.RuleEngine 实现）
│   ├── exec/                # 执行入口（SQL/NoSQL/Search 三路分发）
│   ├── ticket/              # 工单审批流
│   ├── audit/               # 审计日志
│   ├── escalation/          # 提权
│   ├── webhook/             # Webhook 通知
│   ├── project/             # 项目管理
│   ├── user/                # 用户管理
│   ├── role/                # 角色管理
│   ├── permission/          # 权限码管理
│   ├── role_permission/     # 角色-权限码关联
│   ├── user_role/           # 用户-角色关联
│   ├── auth/                # 认证 + 权限检查中间件
│   ├── snippet/             # 代码片段收藏（按数据源类型分类）
│   ├── poolstats/           # 连接池监控
│   └── stats/               # Dashboard 统计
└── pkg/                     # 公共库
    ├── pipeline/            # 指令管道（核心架构）
    │   ├── instruction.go   # Instruction + OpType + 类型映射表
    │   ├── pipeline.go      # Pipeline.Classify + 接口定义
    │   └── parser/          # 各类 Parser 实现
    │       ├── sql.go       # SQL Parser（覆盖 mysql/pg/tidb/oceanbase）
    │       └── redis.go     # Redis Parser
    ├── dbpool/              # 连接池（SQL + Redis 双池隔离）
    ├── database/            # GORM 初始化
    ├── httpx/               # HTTP 工具（JWT、Response、Recover）
    ├── idutil/              # UUID v7 生成
    ├── password/            # 密码加密
    ├── authctx/             # 请求上下文工具
    └── exportx/             # 导出工具
```

## 每个 feature 包的标准文件

```
features/xxx/
├── model.go      # 数据模型（GORM struct + 常量）
├── dto.go        # 请求体/响应体 + ToDTO 转换
├── repo.go       # 数据库操作（增删改查）
├── service.go    # 业务逻辑
├── handler.go    # HTTP handler（Echo Context 处理）
└── router.go     # 路由注册（RegisterRoutes 函数）
```

## 常见扩展场景

### 场景 1：添加一种新数据源类型（如 Kafka）

详见 `docs/database-extension.md`，核心步骤：

**SQL 类型** → 1 个 `pkg/driver/xxx.go` 文件，实现 `SQLConnector` 接口，`init()` 注册
**NoSQL 类型** → 1 个 driver 文件 + 1 个 parser 文件 + exec/ticket 各加 2 行 case
**Search 类型** → 同上

> 工单、审计、Webhook、权限体系全部自动适配。

### 场景 2：添加自定义规则

不需要改代码，通过 seed SQL 或 API 直接添加：

```sql
INSERT INTO datasource_rule (id, name, type_group, type_scope, category, pattern, enabled, priority)
SELECT 'uuid', 'MyRule', 'sql', '', 'dangerous', '^\s*KILL\b', 1, 50;
```

规范：
- 双字段匹配：`type_group`（大类）+ `type_scope`（小类），都留空则匹配全部
- `type_group="" AND type_scope=""`   → 匹配所有类型
- `type_group="sql" AND type_scope=""` → 匹配所有 SQL 类型
- `type_group="" AND type_scope="mysql"` → 仅匹配 MySQL
- `category` 可选值：`read` / `write` / `dangerous`
- `priority` 越小越优先，第一条件命中即生效

### 场景 3：为已有数据源类型添加新命令的 Parser 支持

**SQL Parser** — 编辑 `pkg/pipeline/parser/sql.go` 的 `sqlKeywordPatterns` 表：
```go
{`^\s*EXEC\b`,     pipeline.OpWrite,     "EXEC"},
{`^\s*MERGE\b`,    pipeline.OpDangerous, "MERGE"},
```

**Redis Parser** — 编辑 `pkg/pipeline/parser/redis.go` 的 `redisCommands` 表：
```go
"COPY":     {pipeline.OpWrite, false, "复制 key"},
"RESTORE":  {pipeline.OpWrite, false, "恢复 key"},
```

### 场景 4：实现资源级权限校验

实现 `pipeline.PermissionChecker` 接口并注入：
```go
type MyChecker struct{}
func (c *MyChecker) Check(ctx context.Context, userID string, inst pipeline.Instruction) (bool, error) {
    // 从 inst.Raw 或 inst.Args 解析资源
    // 查自己的权限表，决定 true/false
    return true, nil
}

// 在 newPipeline() 中
pipe.SetPermissionChecker(&MyChecker{})
```

## 启动与调试

```bash
./dev.sh start    # 启动后端(8080) + 前端(5173)
./dev.sh stop     # 停止全部
./dev.sh server   # 仅启动后端
./dev.sh web      # 仅启动前端
```

- 开发数据库：SQLite 文件 `server/jerry.db`（重建则删文件）
- 所有路由表在启动时打印
- seed 文件在 `seeds/sql/required/`，按文件名编号顺序执行，幂等
- 通过 `curl` 验证接口：先 `POST /api/v1/auth/login` 获取 token

## 关键约定

- 每个 feature 包独立，不交叉引用其他 feature（通过全局变量 `global.DB` 操作数据库）
- 不创建 interface 除非有多个实现（如 Parser、Pipeline）
- 错误信息中文/英文都可以，保持一致就行
- 所有 GORM 操作传入 `context`：`global.DB.WithContext(ctx)`
