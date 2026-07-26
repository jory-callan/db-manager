# 其他接口

## Dashboard 统计

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/dashboard/stats` | 仅登录 | 首页统计数据 |

逻辑：返回项目数、数据源数、工单数、审计数等总览统计。

---

## 连接池监控

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/pool-stats` | 仅登录 | 所有连接池状态 |
| GET | `/api/v1/pool-stats/:id` | 仅登录 | 指定连接池详情 |

返回每个连接池的活跃连接数、空闲连接数、等待队列等信息。

---

## 代码片段

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/snippets` | 仅登录 | 全部代码片段（可筛选 datasource_type）|
| GET | `/api/v1/snippets/my` | 仅登录 | 我的代码片段 |
| POST | `/api/v1/snippets` | 仅登录 | 添加代码片段 |
| PUT | `/api/v1/snippets/:id` | 仅登录 | 编辑代码片段 |
| DELETE | `/api/v1/snippets/:id` | 仅登录 | 删除代码片段 |

### POST /api/v1/snippets

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 片段名称 |
| content | string | 是 | 代码内容 |
| datasource_type | string | 是 | 数据源类型（`mysql`/`redis`/`mongo`/`es`/`other`）|

---

## 健康检查

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/health` | 无 | 健康检查 |

响应：`"ok"`（纯文本 text/plain）。
