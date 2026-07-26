# 数据源管理

## 接口一览

| 方法 | 路径 | 权限码 | 说明 |
|------|------|--------|------|
| GET | `/api/v1/datasources` | `db:datasource:view` | 数据源列表 |
| GET | `/api/v1/datasources/:id` | `db:datasource:view` | 数据源详情 |
| POST | `/api/v1/datasources` | `db:datasource:create` | 创建数据源 |
| PUT | `/api/v1/datasources/:id` | `db:datasource:edit` | 编辑数据源 |
| POST | `/api/v1/datasources/:id/test` | `db:datasource:edit` | 测试连接 |
| DELETE | `/api/v1/datasources/:id` | `db:datasource:delete` | 删除数据源 |
| POST | `/api/v1/datasources/batch-delete` | `db:datasource:delete` | 批量删除数据源 |
| GET | `/api/v1/datasource-types` | 仅登录 | 数据源类型列表（按分组）|

---

## 数据源类型系统

数据源有两个层级：

| 字段 | 说明 | 示例 |
|------|------|------|
| `type` | 具体类型 | `mysql` / `postgresql` / `tidb` / `redis` / `mongo` / `es` |
| `type_group` | 类型分组（自动填充）| `sql` / `nosql` / `search` / `mq` |

创建数据源时只需传 `type`，后端根据内置映射自动填充 `type_group`。

### GET /api/v1/datasource-types

返回所有支持的数据源类型，按分组组织。前端用于渲染下拉选择。

```json
{
  "groups": [
    {
      "group": "sql",
      "label": "SQL 数据库",
      "types": [
        { "type": "mysql", "label": "MySQL" },
        { "type": "postgresql", "label": "PostgreSQL" },
        { "type": "tidb", "label": "TiDB" }
      ]
    },
    {
      "group": "nosql",
      "label": "NoSQL",
      "types": [
        { "type": "redis", "label": "Redis" },
        { "type": "mongo", "label": "MongoDB" }
      ]
    }
  ]
}
```

## POST /api/v1/datasources

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 数据源名称 |
| type | string | 是 | 数据库类型：mysql / postgresql / redis / mongo 等 |
| host | string | 是 | 主机地址 |
| port | int | 是 | 端口 |
| username | string | 是 | 连接用户名 |
| password | string | 是 | 连接密码 |
| remark | string | 否 | 备注 |

逻辑：

1. 校验必填字段
2. `type_group` 根据内置映射自动填充
3. 只存入元数据库，**不会在创建时建立连接池**（连接池懒加载）
4. 密码通过统一入口加密存储

## POST /api/v1/datasources/:id/test

逻辑：使用临时连接（不进入连接池）测试连通性，测试完立即关闭。

## DELETE /api/v1/datasources/:id

逻辑：

1. 软删除数据源
2. 关闭该数据源的连接池（如果存在）
3. 已引用该数据源的项目变为无效（业务层处理）

## POST /api/v1/datasources/batch-delete

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | string[] | 是 | 要删除的数据源 ID 列表 |

逻辑：对列表中的每个 ID 执行与单条 DELETE 相同的逻辑。
