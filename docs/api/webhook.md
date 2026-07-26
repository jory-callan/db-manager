# Webhook 配置

## 接口一览

| 方法 | 路径 | 权限码 | 说明 |
|------|------|--------|------|
| GET | `/api/v1/webhooks` | `db:webhook:manage` | Webhook 列表 |
| POST | `/api/v1/webhooks` | `db:webhook:manage` | 创建 Webhook |
| PUT | `/api/v1/webhooks/:id` | `db:webhook:manage` | 编辑 Webhook |
| GET | `/api/v1/webhooks/:id/delivery-logs` | `db:webhook:manage` | 投递日志 |

## POST /api/v1/webhooks

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Webhook 名称 |
| url | string | 是 | 回调地址 |
| events | string[] | 是 | 订阅事件列表 |
| is_active | bool | 否 | 默认 true |

### 支持的事件

| 事件 | 触发时机 |
|------|----------|
| ticket.created | 工单创建 |
| ticket.approved | 审批通过 |
| ticket.partially_approved | all 模式下部分通过 |
| ticket.rejected | 审批拒绝 |
| ticket.executed | 自动执行成功 |
| ticket.execution_failed | 自动执行失败 |
| ticket.urged | 工单催办 |
| escalation.created | 提权申请 |
| escalation.approved | 提权通过 |
| escalation.rejected | 提权拒绝 |
