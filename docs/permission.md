# 权限架构

> 本文档描述 Jerry DB Manager 的权限模型设计。

## 权限模型

### 核心思想

**权限码三段式 + 段级通配匹配，无双层角色模型。**

### 数据表（5 张）

```sql
-- 1. 用户表（纯用户信息，无 role_code）
sys_user (
  id           VARCHAR(32) PRIMARY KEY,  -- UUID v7 无短横线
  username     VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  nickname     VARCHAR(64),
  status       VARCHAR(32) DEFAULT 'active',
  created_at   DATETIME,
  updated_at   DATETIME,
  deleted_at   DATETIME
)

-- 2. 角色表
sys_role (
  id          VARCHAR(32) PRIMARY KEY,  -- UUID v7 无短横线
  code        VARCHAR(64) NOT NULL UNIQUE,
  name        VARCHAR(128) NOT NULL,
  description TEXT,
  is_system   BOOL    DEFAULT FALSE,
  created_at  DATETIME,
  updated_at  DATETIME,
  deleted_at  DATETIME
)

-- 3. 权限码表
sys_permission (
  id          VARCHAR(32) PRIMARY KEY,
  code        VARCHAR(128) NOT NULL UNIQUE,
  name        VARCHAR(128) NOT NULL,
  description TEXT,
  module      VARCHAR(64),
  is_system   BOOL    DEFAULT FALSE,
  created_at  DATETIME,
  updated_at  DATETIME,
  deleted_at  DATETIME
)

-- 4. 用户-角色关联
sys_user_role (
  id          VARCHAR(32) PRIMARY KEY,
  user_id     VARCHAR(32) NOT NULL,
  role_id     VARCHAR(32) NOT NULL,
  created_at  DATETIME,
  updated_at  DATETIME,
  deleted_at  DATETIME
)

-- 5. 角色-权限码关联
sys_role_permission (
  id            VARCHAR(32) PRIMARY KEY,
  role_id       VARCHAR(32) NOT NULL,
  permission_id VARCHAR(32) NOT NULL,
  created_at    DATETIME,
  updated_at    DATETIME,
  deleted_at    DATETIME
)
```

### 权限码设计（三段式）

```
模块:功能:动作
```

#### 标准动作动词

| 动作 | 含义 | 覆盖 API |
|------|------|----------|
| `view` | 查看/读取 | GET 列表 + 详情 |
| `create` | 创建 | POST |
| `edit` | 编辑/更新 | PUT + 配置类操作 |
| `delete` | 删除 | DELETE |

非标准动作按实际功能命名（如 `execute_sql`、`approve`）。

#### 完整权限码列表

```
# 系统（保留）
*                                    # 超级管理员，匹配全部

# 平台管理
platform:user:manage                 # 用户管理：CRUD + 角色分配

# 项目
db:project:view                      # 项目查看：列表、详情、成员
db:project:create                    # 项目创建
db:project:edit                      # 项目编辑：配置、成员管理
db:project:delete                    # 项目删除
db:project:execute_sql               # 执行 SQL
db:project:escalation                # 提权管理：审批/拒绝

# 工单
db:ticket:approve                    # 工单审批

# 数据源
db:datasource:view                   # 数据源查看
db:datasource:create                 # 数据源创建
db:datasource:edit                   # 数据源编辑+测试连接
db:datasource:delete                 # 数据源删除

# SQL 规则
db:rule:view                         # 规则查看
db:rule:create                       # 规则创建
db:rule:edit                         # 规则编辑

# 审计
db:audit:view                        # 审计日志查看

# Webhook
db:webhook:manage                    # Webhook 管理
```

| 权限码 | 含义 |
|---|---|
| `*` | 超级管理员，匹配全部 |
| `db:*` | db 模块管理员，匹配所有 `db:xxx:xxx`（等价 `db:*:*`）|
| `db:datasource:*` | db 数据源子模块管理员，匹配所有 `db:datasource:xxx` |
| `db:*:view` | db 模块下所有资源的查看操作，匹配 `db:project:view`、`db:datasource:view` |
| `db:sql:read` | 精确匹配 |

> ⚠️ `*:view`、`*:*:*` 等以 `*` 开头的模式**被禁止**，不会匹配任何权限码。

### 匹配算法

```go
func match(pattern, target string) bool {
    // 超级管理员，匹配全部
    if pattern == "*" {
        return true
    }

    // 不含通配符 → 精确匹配
    starIdx := strings.IndexByte(pattern, '*')
    if starIdx < 0 {
        return pattern == target
    }

    // * 在段首 → 禁止（如 "*:xxx"、"*:xxx:xxx"）
    if starIdx == 0 {
        return false
    }

    // 拆分为前缀和后缀
    prefix := strings.TrimSuffix(pattern[:starIdx], ":")
    suffix := pattern[starIdx+1:]

    if suffix == "" {
        // "xx:*" 模式 → 以 prefix: 开头即可（等价 xx:*:*）
        return strings.HasPrefix(target, prefix+":") || target == prefix
    }

    // "xx:*:zz" 模式 → 前缀 + xxx + 后缀
    return strings.HasPrefix(target, prefix+":") && strings.HasSuffix(target, suffix)
}
```

### 权限码 ↔ API

权限码和 API 接口是 **1 对多**关系。例如 `db:project:access` 覆盖：

- GET /db/projects
- GET /db/projects/:id
- GET /db/projects/:id/members
- GET /db/projects/:id/tickets
- GET /db/tickets/:id

### 系统默认数据

#### 权限码

| 权限码 | 说明 | 模块 |
|---|---|---|
| `*` | 超级管理员（系统保留） | system |
| `platform:user:manage` | 用户管理 | platform |
| `platform:module:manage` | 模块管理 | platform |
| `db:project:access` | 项目访问 | db |
| `db:project:execute_sql` | 执行 SQL | db |
| `db:project:manage` | 项目管理 | db |
| `db:project:manage_escalation` | 提权管理 | db |
| `db:ticket:approve` | 工单审批 | db |
| `db:audit:view` | 审计查看 | db |
| `db:rule:manage` | 规则管理 | db |
| `db:datasource:manage` | 数据源管理 | db |

#### 角色

| 角色 | 说明 | 绑定的权限码 |
|---|---|---|
| `platform_admin` | 平台管理员 | `*` |
| `sql_admin` | SQL 管理员 | `db:*` |
| `sql_dev` | SQL 开发 | `db:project:access`, `db:project:execute_sql` |
| `sql_viewer` | SQL 查看 | `db:project:access` |

### 中间件模式

```go
// 入口权限检查（中间件）
func RequirePermission(code string) echo.MiddlewareFunc { ... }

// 业务特判（handler 内）
func Execute(c echo.Context) error {
    // 中间件已确保有 db:project:execute_sql
    // 业务特判：owner / 提权等
    if isProjectOwner(userID, projectID) {
        return executeAndAudit(...)
    }
    if hasActiveEscalation(userID, projectID) {
        return executeAndAudit(...)
    }
    // ...
}
```

## 迁移说明

当前为 v0.3.0 权限码模型实现，从 v0.2.0 的 role_code 双层模型迁移而来：

- 移除 `users.role_code` 字段
- 新增 `permissions`、`roles`、`user_roles`、`role_permissions` 4 张表
- 所有路由改用权限码中间件
- 用户角色管理通过 UI 配置（无硬编码）
