# 审计日志

## 接口一览

| 方法 | 路径 | 权限码 | 说明 |
|------|------|--------|------|
| GET | `/api/v1/audits` | `db:audit:view` | 审计列表（分页）|

## GET /api/v1/audits

查询参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| action | string | 按操作类型过滤 |
| classification | string | 按分类过滤（read / write / dangerous / unknown）|
| actor_id | string | 按操作用户过滤 |
| project_id | string | 按项目过滤 |
| datasource_id | string | 按数据源过滤 |
| status | string | 按状态过滤（success / failed / rejected）|
| start_time | string | 开始时间 |
| end_time | string | 结束时间 |
| page | int | 页码 |
| page_size | int | 每页条数 |

### 审计日志字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | UUID v7 |
| actor_id | string | 操作人用户 ID |
| project_id | string | 关联项目 ID |
| datasource_id | string | 关联数据源 ID |
| action | string | 操作类型（execute / escalated_execute / ticket_execute / approve / reject 等）|
| raw_text | string | 原始输入文本（可模糊搜索）|
| instruction_json | string | JSON 序列化的 `[]Instruction`（结构化展示，包含 type / command / op_type 等信息）|
| classification | string | 分类: read / write / dangerous / unknown |
| status | string | 状态: success / failed / rejected |
| duration_ms | int | 执行耗时（毫秒）|
| error_message | string | 错误信息 |
| ticket_id | string | 关联工单 ID |
| ip | string | 请求来源 IP |
| created_at | string | 创建时间 |

### 审计记录的 action 类型

| action | 触发时机 |
|--------|----------|
| execute | SQL/Redis 执行 |
| escalated_execute | 提权执行 |
| ticket_execute | 工单自动执行 |
| approve / reject | 工单审批 |

注意：审计只存元信息（操作人、指令、耗时、状态、IP），**不存结果集**。

### instruction_json 示例

```json
[
  {
    "type": "mysql",
    "type_group": "sql",
    "command": "SELECT",
    "args": ["*", "FROM", "users"],
    "raw": "SELECT * FROM users",
    "op_type": "read"
  },
  {
    "type": "redis",
    "type_group": "nosql",
    "command": "SET",
    "args": ["mykey", "myval"],
    "raw": "SET mykey myval",
    "op_type": "write"
  }
]
```
