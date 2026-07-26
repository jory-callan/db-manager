# 工单审批流

> 写操作的审批单元。支持 SQL / Redis 等所有数据源类型。提交 → 审批 → 自动执行。

## 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/tickets` | 创建工单 |
| GET | `/api/v1/tickets` | 工单列表 |
| GET | `/api/v1/tickets/:id` | 工单详情 |
| POST | `/api/v1/tickets/:id/approve` | 审批通过 |
| POST | `/api/v1/tickets/:id/reject` | 拒绝 |
| POST | `/api/v1/tickets/:id/urge` | 催办 |
| POST | `/api/v1/tickets/:id/resubmit` | 重新提交已拒绝的工单 |

---

## 工单状态机

```
pending ─────────────────────────────┐
   │                                  │ 主动撤销（暂无接口，可扩展）
   ↓                                  ↓
approved ───→ executing ──→ executed   rejected
                   │
                   └──→ execute_failed
```

| 状态 | 含义 |
|------|------|
| pending | 待审批 |
| approved | 审批通过，等待自动执行 |
| executing | 正在执行 |
| executed | 执行成功 |
| execute_failed | 执行失败 |
| rejected | 已拒绝 |

## 审批模式

每个项目配置一种审批模式：

- **any_one**（默认）：任一审批人通过即可执行
- **all**：所有审批人都通过才能执行

## 审批人判定

可审批工单的人员包括：
- 在项目成员中拥有 `approver` 或 `admin` 角色的用户
- 拥有 `*` 权限码的用户（系统管理员）
- 同一用户有多个角色时可正常处理

审批人不通过独立的 approver_ids 字段维护，而是统一通过成员管理的角色来管理。

## 完整业务决策树

### 用户提交工单

```
用户提交指令（SQL / Redis 命令）
   ↓
[1] 校验项目、申请人、指令 JSON
   ↓
[2] 创建工单（status=pending），instruction_json 保存指令快照
   ↓
[3] 发送 webhook: ticket.created
```

### 审批人审批

```
审批人调用 approve
   ↓
[1] 工单状态不是 pending？→ 400（已处理过）
   ↓
[2] 已审批过？→ 400（已处理过）
   ↓
[3] 身份检查（满足任一即可）：
   ├─ 是项目成员（approver 或 admin 角色）
   └─ 拥有 * 权限码（超管）
   ↓
[4] 创建审批记录（approval_record）
   ↓
[5] 判断是否执行：
   ├─ any_one 模式 → 第一个审批通过即执行
   ├─ all 模式 → 检查是否所有 approver 已通过
   └─ 不满足 → status 保持 pending，发 partially_approved 通知
   ↓
[6] 满足执行条件：
   ├─ 状态改为 approved
   ├─ 发送 webhook: ticket.approved
   └─ 自动执行指令快照（不依赖人工操作）
```

### 自动执行快照

```
系统自动执行
   ↓
[1] 状态改为 executing
   ↓
[2] 反序列化 instruction_json 为 []Instruction
   ↓
[3] 按指令类型分发：
   ├─ mysql/pg/tidb → dbpool.Get + ExecContext
   ├─ redis         → dbpool.GetRedis + client.Do
   └─ 其他          → 错误
   ↓
[4] 逐条执行
   ├─ 全部成功？→ 状态 executed，写审计，发 webhook
   └─ 有失败？→ 状态 execute_failed，写审计，发 webhook
```

---

## POST /api/v1/tickets

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| project_id | string | 是 | 项目 ID |
| instruction_json | string | 是 | JSON 序列化的 `[]Instruction`（工单提交后锁定为快照）|
| title | string | 否 | 工单标题 |
| description | string | 否 | 工单描述 |

## POST /api/v1/tickets/:id/approve

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment | string | 否 | 审批意见 |

## POST /api/v1/tickets/:id/reject

请求体同 approve。

## POST /api/v1/tickets/:id/urge

逻辑：
1. 只有申请人可以催办
2. 催办间隔至少 30 分钟
3. 记录催办事件

## POST /api/v1/tickets/:id/resubmit

逻辑：
1. 只有申请人可以重新提交
2. 原工单必须是 rejected 状态
3. 继承原工单的项目、审批模式
4. 创建新工单（不修改原工单）

---

## 关键规则

| 规则 | 说明 |
|------|------|
| **指令不可变** | 工单提交时存 instruction_json，审批后只执行快照，不可修改。不再局限于 SQL 字符串，Redis 命令也可通过工单审批执行 |
| **自动执行** | 审批通过后系统自动执行，不依赖人工 |
| **超管兜底** | 拥有 `*` 权限码可绕过审批人列表直接审批 |
| **不跨库事务** | 工单状态和指令执行各自记录，通过审计保证可追踪 |
