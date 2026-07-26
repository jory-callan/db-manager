# 平台管理

所有接口需要 `platform:user:manage` 权限码。包括用户、角色、权限码管理。

---

## 用户管理

### 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users` | 用户列表（分页）|
| POST | `/api/v1/users` | 创建用户 |
| GET | `/api/v1/users/:id` | 用户详情 |
| PUT | `/api/v1/users/:id` | 编辑用户 |
| DELETE | `/api/v1/users/:id` | 删除用户 |
| DELETE | `/api/v1/users/batch` | 批量删除 |

### GET /api/v1/users

查询参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| keyword | string | 模糊匹配 username / nickname / email |
| status | string | 按状态过滤：active / disabled |
| page | int | 页码（默认 1）|
| page_size | int | 每页条数（默认 20，最大 2000）|
| need_count | bool | 是否返回 total |

### POST /api/v1/users

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 最长 64 字符，唯一 |
| password | string | 是 | 至少 6 字符 |
| nickname | string | 否 | 不传则等于 username |
| email | string | 否 | 电子邮箱 |
| phone | string | 否 | 手机号 |
| status | string | 否 | 默认 active，可选 disabled |

逻辑：

1. 校验必填字段 → 400
2. 检查 username 是否已存在 → 409
3. 密码加密存 `password_hash`
4. 自动生成 UUID v7
5. 创建用户，返回完整信息

### PUT /api/v1/users/:id

请求体（全可选）：

| 字段 | 类型 | 说明 |
|------|------|------|
| nickname | string | 新昵称 |
| email | string | 新邮箱 |
| phone | string | 新手机号 |
| status | string | active / disabled |
| password | string | 新密码（≥ 6 字符）|

### DELETE /api/v1/users/:id

逻辑：软删除（gorm.DeletedAt）。

### DELETE /api/v1/users/batch

请求体：`{ "ids": [1, 2, 3] }`

---

## 用户角色管理

### 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users/:userId/roles` | 用户的角色列表 |
| POST | `/api/v1/users/:userId/roles` | 分配角色 |
| DELETE | `/api/v1/users/:userId/roles/:roleId` | 移除角色 |

### POST /api/v1/users/:userId/roles

请求体：`{ "role_id": 1 }`

逻辑：
1. 检查该用户是否已有该角色 → 409
2. 创建关联

---

## 角色管理

### 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/roles` | 角色列表 |
| GET | `/api/v1/roles/:id` | 角色详情 |
| POST | `/api/v1/roles` | 创建角色 |
| PUT | `/api/v1/roles/:id` | 编辑角色 |
| DELETE | `/api/v1/roles/:id` | 删除角色 |

### POST /api/v1/roles

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | 角色标识，唯一 |
| name | string | 是 | 角色名称 |
| description | string | 否 | 描述 |

### PUT /api/v1/roles/:id

请求体（全可选）：`{ name, description }`

---

## 权限码管理

### 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/permissions` | 权限码列表 |
| GET | `/api/v1/permissions/:id` | 权限码详情 |
| POST | `/api/v1/permissions` | 创建权限码 |
| PUT | `/api/v1/permissions/:id` | 编辑权限码 |
| DELETE | `/api/v1/permissions/:id` | 删除权限码 |
| GET | `/api/v1/permissions/:id/bindings` | 权限码绑定关系（被哪些角色/用户使用）|

### GET /api/v1/permissions

查询参数：`module` — 按模块过滤（如 `db`、`platform`）

### POST /api/v1/permissions

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | 权限码标识（如 `db:project:view`），唯一 |
| name | string | 是 | 权限码名称 |
| description | string | 否 | 描述 |
| module | string | 否 | 所属模块（如 `db`、`platform`）|

### PUT /api/v1/permissions/:id

请求体（全可选）：

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 新名称 |
| description | string | 新描述 |
| module | string | 新所属模块 |

逻辑：
1. 校验 id → 400
2. 检查权限码是否存在 → 404
3. 更新字段，返回完整信息

### GET /api/v1/permissions/:id/bindings

返回该权限码被哪些角色引用、以及这些角色下的用户。

响应体：

```json
{
  "permission": {
    "id": "pid001",
    "code": "db:project:view",
    "name": "查看项目",
    "module": "db",
    "is_system": false,
    ...
  },
  "roles": [
    { "id": "rid001", "code": "developer", "name": "开发者" }
  ],
  "users": [
    { "id": "uid001", "username": "zhangsan", "nickname": "张三" }
  ]
}
```

逻辑：
1. 校验 id → 400
2. 检查权限码是否存在 → 404
3. 查询 `role_permission` 表获取直接关联该权限码的角色
4. 查找所有包含 `*` 的权限码模式（如 `*`、`db:*`、`db:*:view`），若通配匹配目标权限码则将对应角色也纳入
5. 遍历所有关联角色，通过 `user_role` 表查出每个角色下的用户（去重）
6. 返回权限码信息 + 角色列表 + 用户列表

> 通配匹配规则详见 [docs/permission.md](../permission.md)。

---

## 角色-权限码关联

### 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/roles/:roleId/permissions` | 查看角色的权限码列表 |
| POST | `/api/v1/roles/:roleId/permissions` | 给角色绑定权限码 |
| DELETE | `/api/v1/roles/:roleId/permissions/:permId` | 移除权限码 |

### POST /api/v1/roles/:roleId/permissions

请求体：`{ "permission_id": 1 }`

逻辑：
1. 检查该角色是否已有该权限码 → 409
2. 创建关联
