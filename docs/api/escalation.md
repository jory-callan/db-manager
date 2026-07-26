# 提权管理

> developer 临时绕过工单直接执行写 SQL 的机制，有 TTL。

## 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/escalations` | 创建提权申请 |
| GET | `/api/v1/escalations` | 提权列表 |
| GET | `/api/v1/escalations/active` | 查看当前用户的活跃提权 |
| PUT | `/api/v1/escalations/:id` | 编辑提权理由（仅 pending + 本人） |
| DELETE | `/api/v1/escalations/:id` | 删除提权申请（仅 pending + 本人） |
| DELETE | `/api/v1/escalations/batch` | 批量删除提权申请（仅 pending + 本人） |
| POST | `/api/v1/escalations/:id/approve` | 审批通过 |
| POST | `/api/v1/escalations/:id/reject` | 拒绝 |

---

## 提权生命周期

```
developer 申请提权
   ↓
status: pending
   ↓
owner / 超管 审批
   ├─ 批准 → status: active（至 expires_at 到期自动 expired）
   └─ 拒绝 → status: rejected
   ↓
active 期间 developer 可调用 `/execute` 绕过工单
   ↓
到期 → status: expired
```

## 关键规则

| 规则 | 说明 |
|------|------|
| 申请资格 | 任何项目成员可申请提权 |
| 审批资格 | 项目成员中拥有 admin 或 approver 角色的用户，或拥有 `*` 权限码的用户，申请人也可自批 |
| 有效期 | 最长 1 年，审批时可通过 duration 参数缩短 |
| 审计 | 每次提权操作都有独立审计记录 |
| 和工单的区别 | 提权适合"紧急、临时、多次操作"，工单适合"单次、需多人审批" |

---

## POST /api/v1/escalations

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| project_id | int | 是 | 项目 ID |
| reason | string | 是 | 提权原因 |

## GET /api/v1/escalations

查询参数：

| 参数 | 说明 |
|------|------|
| scope | all（默认，管理员看全部）/ my（只看自己的）|
| status | pending / approved / rejected / expired |

## POST /api/v1/escalations/:id/approve

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| duration | string | 否 | 时长（如 "2h"、"7d"），不传默认 1 年 |

逻辑：

1. 工单必须是 pending 状态
2. 身份检查：project_owner 或 `*` 超管，或申请人自批
3. 设置有效期（审批时指定的 duration，最长 1 年）
4. 状态改为 approved
5. 发送 webhook: escalation.approved

## PUT /api/v1/escalations/:id

编辑提权申请理由（仅 pending 状态 + 本人）。

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reason | string | 是 | 新的提权理由 |

## DELETE /api/v1/escalations/:id

删除提权申请（仅 pending 状态 + 本人）。软删除。

## DELETE /api/v1/escalations/batch

批量删除提权申请。

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | string[] | 是 | 要删除的提权申请 ID 列表 |

后端过滤：仅删除当前用户、pending 状态的提权申请，不存在的 ID 静默忽略。

## POST /api/v1/escalations/:id/reject

逻辑：同上身份检查，状态改为 rejected。
