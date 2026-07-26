# 指令执行

> 核心功能模块。所有数据源类型（MySQL / PostgreSQL / Redis / MongoDB / ES 等）统一走指令管道。
> 入口不设权限码中间件，权限在 handler 内做业务特判。

---

## 一、接口一览

| 方法 | 路径 | 说明 | 前端用途 |
|------|------|------|---------|
| POST | `/api/v1/execute` | 执行指令（SQL/Redis/Mongo/ES 三路分发）| 点击执行按钮 |
| POST | `/api/v1/execute/escalated` | 以提权方式执行写操作 | 点击提权执行按钮 |
| GET | `/api/v1/projects/:id/meta/databases` | 获取项目的数据库列表 | 数据库树：展开项目→显示库 |
| GET | `/api/v1/projects/:id/meta/tables?database=xxx` | 获取数据库的表列表 | 数据库树：展开库→显示表 |
| GET | `/api/v1/projects/:id/meta/columns?database=xxx&table=yyy` | 获取表的列信息（含类型、备注）| 数据库树：展开表→显示列 + Monaco 联想 |

---

## 二、POST /api/v1/execute — 核心执行接口

### 请求体

```json
{
  "project_id": "019f...",
  "sql": "SELECT * FROM users",
  "database": "mydb",
  "selected_text": ""
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| project_id | string | 是 | 项目 ID |
| sql | string | 是 | 要执行的指令（SQL / Redis 命令 / ES 请求 / Mongo 命令）|
| database | string | 否 | 指定数据库名（仅 SQL 类型有效，Redis/Mongo/ES 忽略）|
| selected_text | string | 否 | 编辑器选中部分执行（非空时覆盖 sql）|

### 响应（读操作成功）

```json
{
  "mode": "direct",
  "execution": {
    "id": "...",
    "status": "success",
    "duration_ms": 150,
    "row_count": 100,
    "affected_rows": 0
  },
  "results": [
    {
      "columns": [
        { "name": "id", "database_type": "INT", "nullable": false },
        { "name": "name", "database_type": "VARCHAR", "nullable": true }
      ],
      "rows": [
        [1, "Alice"],
        [2, "Bob"]
      ],
      "total": 2
    }
  ]
}
```

**注意：**
- `rows` 是 `[][]any`，每行是一个数组，**顺序与 columns 一一对应**
- `columns` 中的 `length` / `precision` / `scale` 因驱动而异（MySQL 不返回 length，PG 返回），不适用时 JSON 中不输出该字段
- `comment` 仅在 `GET /meta/columns` 接口返回，执行结果不附带

### 前端处理逻辑

```
收到 ExecuteResponse
  │
  ├─ mode === "ticket"
  │   → 弹出工单对话框，让用户填写原因后提交
  │
  ├─ mode === "direct" 且 results 有内容
  │   → 取 results[i].columns 作为表头
  │   → 取 results[i].rows 作为行数据（列下标对应）
  │   → 渲染到 VirtualResultTable
  │
  └─ mode === "direct" 且 results 无内容（DML 操作）
      → 显示 "执行成功，影响 N 行"
```

### 后端决策流程（了解即可）

```
请求 → 检查 project 权限 → Pipeline.Classify()
  ├─ dangerous → HTTP 403 "dangerous operation rejected"
  ├─ unknown   → HTTP 403 "unrecognizable statement"
  ├─ write     → HTTP 403 "写操作已拒绝，请提交工单或提权执行"
  └─ read      → 按 type_group 分发执行
       ├─ sql    → *sql.DB + SQLConnector + ScanRows
       ├─ nosql  → Redis/MongoDB 各自驱动
       └─ search → ES 的 Transport.Perform
```

---

## 三、POST /api/v1/execute/escalated — 提权执行

与 `/execute` 完全相同的请求/响应格式，区别：

- 如果是写操作，**不走创建工单**，而是检查用户当前项目是否有**活跃提权**
- **超管（拥有 `*` 权限码）**：跳过提权检查，直接执行
- 普通用户：有活跃提权 → 直接执行；无 → HTTP 403

前端使用场景：用户点击「提权执行」按钮。

---

## 四、元数据接口（数据库树 + 联想用）

这三个接口供前端左侧数据库树面板和 Monaco 编辑器自动联想使用。

### 4.1 GET /projects/:id/meta/databases

获取项目的可访问数据库列表。

**权限：** 仅项目成员

**响应：**

```json
{
  "databases": ["mydb", "testdb", "analytics"]
}
```

**前端用途：** 数据库树第一层。点击项目时调用，展开显示数据库名。

---

### 4.2 GET /projects/:id/meta/tables?database=xxx

获取指定数据库的表列表。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| database | string | 是 | 数据库名 |

**权限：** 仅项目成员，且 database 必须在项目 scope 内

**响应：**

```json
{
  "tables": [
    { "database": "mydb", "table": "users", "type": "TABLE" },
    { "database": "mydb", "table": "orders", "type": "TABLE" },
    { "database": "mydb", "table": "user_order_view", "type": "VIEW" }
  ]
}
```

**前端用途：** 数据库树第二层。点击数据库时调用，展开显示表名。
`type` 字段可用于图标区分（TABLE=表格图标，VIEW=视图图标）。

---

### 4.3 GET /projects/:id/meta/columns?database=xxx&table=yyy

获取指定表的列信息（含类型、长度、备注）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| database | string | 是 | 数据库名 |
| table | string | 是 | 表名 |

**权限：** 仅项目成员，且 database 必须在项目 scope 内

**响应：**

```json
{
  "columns": [
    {
      "name": "id",
      "database_type": "INT",
      "length": null,
      "precision": null,
      "scale": null,
      "nullable": false,
      "comment": "主键 ID"
    },
    {
      "name": "name",
      "database_type": "VARCHAR",
      "length": 255,
      "nullable": true,
      "comment": "用户名"
    },
    {
      "name": "hobby",
      "database_type": "VARCHAR",
      "length": 100,
      "nullable": true,
      "comment": "爱好"
    },
    {
      "name": "created_at",
      "database_type": "DATETIME",
      "nullable": false,
      "comment": "创建时间"
    }
  ]
}
```

**接口说明：**
- `comment` 字段来自数据库表的列注释（MySQL 的 `COLUMN_COMMENT`，PostgreSQL 的 `column_comment`）
- 不适用时 `length`/`precision`/`scale` 返回 `null`，前端判 `!= null` 决定是否展示
- MySQL 驱动不报告 VARCHAR 长度（详见"限制说明"），但本接口直接查 INFORMATION_SCHEMA，保证有长度

**前端用途：** 两个场景

| 场景 | 调用时机 | 展示 |
|------|---------|------|
| 数据库树展开列 | 用户点击表名展开时 | 表下显示列列表，格式：`name  (VARCHAR)  — 注释说明` |
| Monaco 编辑器自动联想 | 用户输入 SQL 时 | 根据表名联想对应的列名 |

---

## 五、执行成功的列信息与 meta/columns 的区别

| | 执行结果中的 columns | GET /meta/columns |
|---|-------------------|-------------------|
| 来源 | Go 标准库 `rows.ColumnTypes()` | INFORMATION_SCHEMA 查询 |
| 含 comment | ❌ 无 | ✅ 有 |
| 含 length | ⚠️ 因驱动而异 | ✅ 保证有 |
| 来源时机 | 每次执行自动附带 | 用户主动展开表时请求 |

**建议：** 执行结果的列信息用于表格渲染，列备注信息通过展开数据库树获取。

---

## 六、已知限制

| 限制 | 原因 |
|------|------|
| MySQL 执行结果不返回 VARCHAR length | go-sql-driver 把 ColumnTypeLength() 注释了 |
| 执行结果不返回 comment | `rows.ColumnTypes()` 无此 API |
| `length` 可能为 null | 非变长类型（INT/DATETIME）没有长度概念 |
