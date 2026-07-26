# 项目管理

## 接口一览

| 方法 | 路径 | 权限码 | 说明 |
|------|------|--------|------|
| GET | `/api/v1/projects` | `db:project:view` | 项目列表（分页）|
| GET | `/api/v1/projects/:id` | `db:project:view` | 项目详情 |
| GET | `/api/v1/projects/:id/members` | `db:project:view` | 项目成员列表 |
| POST | `/api/v1/projects` | `db:project:create` | 创建项目 |
| PUT | `/api/v1/projects/:id` | `db:project:edit` | 编辑项目 |
| PUT | `/api/v1/projects/:id/members` | `db:project:edit` | 更新项目成员 |
| DELETE | `/api/v1/projects/:id` | `db:project:delete` | 删除项目 |
| POST | `/api/v1/projects/batch-delete` | `db:project:delete` | 批量删除项目 |

---

## 关键概念

| 概念 | 说明 |
|------|------|
| 项目（Project） | 访问边界，绑定一个数据源 + `scope` 限定资源范围 |
| 项目成员（Member） | 项目内的角色：viewer / developer / approver / admin |
| 审批模式 | any_one（任一通过即可执行）/ all（全部通过）|
| 审批人管理 | 通过成员管理给用户分配 Approver 或 Admin 角色即为审批人，不再使用独立的 approver_ids 字段 |
| 资源范围（scope） | JSON 字符串，语义按数据源类型自动适配：SQL→数据库名列表，Redis→db number，Mongo→database，ES→index pattern |

---

## POST /api/v1/projects

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 项目名称 |
| datasource_id | string | 是 | 绑定的数据源 ID |
| scope | string[] | 否 | 资源范围列表（逗号分隔存储）|
| approval_mode | string | 否 | 默认 any_one |
| webhook_ids | string[] | 否 | Webhook ID 列表 |

## PUT /api/v1/projects/:id

逻辑：只更新请求体中传了的字段（部分更新）。

## PUT /api/v1/projects/:id/members

请求体：`{ "members": [{ "user_id": 1, "role": "developer" }, ...] }`

逻辑：**全量替换** — 删除原有成员，写入新成员列表。不是增量追加。

支持的角色：`viewer`（只读）、`developer`（读写+工单）、`approver`（审批人）、`admin`（项目管理，拥有审批和管理权限）
