# API 文档

> 前端对接指南。每个模块一个独立文件，复杂业务逻辑有完整说明。

---

## 目录

| 文件 | 模块 |
|------|------|
| [auth.md](auth.md) | 认证（登录、获取用户、修改个人资料）|
| [platform.md](platform.md) | 平台管理（用户、角色、权限码管理）|
| [datasource.md](datasource.md) | 数据源管理 |
| [project.md](project.md) | 项目管理 |
| [sql-execution.md](sql-execution.md) | 指令执行（含完整业务决策树，SQL/NoSQL/Search）|
| [ticket.md](ticket.md) | 工单审批流（含完整状态机说明）|
| [escalation.md](escalation.md) | 提权管理 |
| [rule.md](rule.md) | 数据源分类规则管理 |
| [audit.md](audit.md) | 审计日志 |
| [webhook.md](webhook.md) | Webhook 配置 |
| [other.md](other.md) | 其他接口（统计、连接池、代码片段）|

> 后端开发指引见 [docs/backend-development.md](../backend-development.md)。

---

## 全局响应格式

成功：

```json
{
  "code": 0,
  "msg": "success",
  "data": { ... }
}
```

失败：

```json
{
  "code": 401,
  "msg": "unauthorized"
}
```

分页（请求带 `page` + `page_size` + `need_count`）：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 前端约定

> 部分细节参考 `web/CONVENTIONS.md`

- API 基础路径：`/api/v1`
- 登录后请求头带 `Authorization: Bearer <token>`
- 401 自动跳转 `/login`
- 分页查询用 `keepPreviousData`
- 分页参数：`page`、`page_size`（最大 2000）、`need_count`
- 响应字段 `code` 为 0 表示成功，非 0 表示失败

---

## 全局错误码

| code | HTTP | 含义 |
|------|------|------|
| 0 | 200 | 成功 |
| 400 | 400 | 请求参数错误（msg 返回具体原因）|
| 401 | 401 | 未登录 / token 过期 / 无效 |
| 403 | 403 | 无权限码（或业务层拒绝） |
| 404 | 404 | 资源不存在 |
| 409 | 409 | 资源冲突（用户名/权限码/角色重复） |
| 422 | 422 | 业务逻辑拒绝（如高危 SQL 被拦截） |
| 500 | 500 | 服务器内部错误 |

### 分页列表的统一筛选协议

所有分页列表接口（`GET /api/v1/{resource}`）支持统一的动态筛选参数。

**分页参数**（固定）：

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `page` | int | `1` | 页码 |
| `page_size` | int | `20` | 每页条数，最大 2000 |
| `need_count` | bool | `true` | 是否返回总数 `total` |

**筛选参数**（动态，以每个接口实际支持为准）：

筛选参数名就是数据库字段名（或约定别名），筛选值通过**操作符前缀**约定来指定匹配方式：

| 前缀 | 含义 | 示例 | 等效 SQL |
|------|------|------|----------|
| 无前缀 | 精确匹配（向后兼容） | `status=active` | `status = 'active'` |
| `=` | 精确匹配（显式） | `status==active` | `status = 'active'` |
| `=～` / `=~` | LIKE 模糊匹配 | `name==～*test*` | `name LIKE '%test%'` |
| `=[` | IN 列表 | `project_id==[a,b,c]` | `project_id IN ('a','b','c')` |
| `=gte:` | 大于等于 | `duration_ms==gte:100` | `duration_ms >= 100` |
| `=lte:` | 小于等于 | `duration_ms==lte:50` | `duration_ms <= 50` |
| `=gt:` | 大于 | `count==gt:10` | `count > 10` |
| `=lt:` | 小于 | `count==lt:5` | `count < 5` |
| `=between:` | 范围 | `created_at==between:2024-01-01,2024-12-31` | `created_at BETWEEN '2024-01-01' AND '2024-12-31'` |
| `_all` 或空 | 跳过该筛选 | `status=_all` | 无过滤 |

> **注意**：具体某个字段支持哪些操作符，取决于后端的 `columnMap` 配置。`=～` 中的波浪号是全角字符（U+FF5E），也兼容 ASCII `=~`。

**示例：**

```
GET /api/v1/users?page=1&page_size=20&need_count=false&status==active&name__contains==～*admin*
```

- 分页：第 1 页，每页 20 条，不返回总数
- 筛选：`status` 精确匹配 `active`，`name` 模糊匹配包含 `admin` 的记录

### 常见 error msg

| msg | 说明 |
|-----|------|
| `invalid param` / `invalid id` | 参数校验不通过 |
| `unauthorized` | token 缺失或无效 |
| `forbidden` | 无对应权限码 |
| `NOT_PROJECT_MEMBER` | 不是项目成员（业务层拒绝）|
| `NOT_PROJECT_OWNER` | 不是项目 owner |
| `xxx not found` | 资源不存在 |
| `xxx already exists` | 资源名重复 |
| `HIGH_RISK_BLOCKED` | 高危 SQL 被拦截 |

---

## 认证方式

所有受保护接口请求头需携带：

```
Authorization: Bearer <token>
```

token 通过 `POST /api/v1/auth/login` 获取。

---

## 权限说明

两个层面的权限检查：

1. **路由中间件层**：检查用户是否有指定权限码（如 `db:project:view`）。没有直接 403。
2. **业务层（handler/service 内）**：检查项目成员身份、owner 身份、提权状态等。

详见 [docs/permission.md](../permission.md)。
