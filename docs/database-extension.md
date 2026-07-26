# 数据源类型扩展指南

> 本文档说明如何为 Jerry DB Manager 添加新的数据源类型支持。

## 架构概述

系统有三层扩展点，从上到下：

```
┌─ 调度层 ──────────────────────────────┐
│  features/exec/    — Execute() 按 type_group 分发 │
│  features/ticket/  — ExecuteTicket() 按 type_group 分发 │
├─ 分类层 ──────────────────────────────┤
│  pkg/pipeline/parser/ — 各 Parser 按 type 注册     │
│  pkg/pipeline/instruction.go — 类型映射表           │
├─ 连接层 ──────────────────────────────┤
│  pkg/driver/ — SQLConnector(注册表) + 辅助函数      │
│  pkg/dbpool/  — 四池隔离(SQL/Redis/Mongo/ES)       │
└──────────────────────────────────────────────────┘
```

---

## 扩展步骤

### 加一个新 SQL 类型（如 SQL Server）

**只需 1 个文件：**

```
pkg/driver/sqlserver.go
```

```go
package driver

import "database/sql"

func init() {
    RegisterSQLConnector("sqlserver", &mssqlConnector{})
}

type mssqlConnector struct{}

func (m *mssqlConnector) DSN(cfg ConnConfig) (string, string) {
    dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
        cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
    return "sqlserver", dsn
}

func (m *mssqlConnector) SetDatabase(ctx, db, database) error {
    _, err := db.ExecContext(ctx, "USE ["+database+"]")
    return err
}

func (m *mssqlConnector) ListDatabases(ctx, db) ([]string, error) { ... }
func (m *mssqlConnector) ListTables(ctx, db, database) ([]TableInfo, error) { ... }
func (m *mssqlConnector) ListColumns(ctx, db, database, table) ([]ColumnInfo, error) { ... }
func (m *mssqlConnector) DefaultPort() int { return 1433 }
```

**零改动已有代码。** `executeSQLRead()` 对所有 SQL 类型自动通用，`ScanRows()` 自动读取列类型信息。

---

### 加一个新 NoSQL 类型（如 Neo4j）

**需要 3 步：**

#### 1. `pkg/driver/neo4j.go` — 连接辅助函数

```go
package driver

// ConnectNeo4j 创建 Neo4j 连接
func ConnectNeo4j(cfg ConnConfig) (*neo4j.DriverWithContext, error) {
    // ...
}
```

#### 2. `pkg/pipeline/parser/neo4j.go` — Parser

```go
package parser

type Neo4jParser struct{}
func (p *Neo4jParser) Types() []pipeline.DataSourceType {
    return []pipeline.DataSourceType{pipeline.DsNeo4j}
}
func (p *Neo4jParser) Parse(_ context.Context, dsType, raw string) ([]pipeline.Instruction, error) {
    // 解析 Cypher 语法，分类 MATCH=read / CREATE=write / DELETE=dangerous
}
```

#### 3. `features/exec/service.go` — 注册 + 执行

`s` 函数里注册 parser：
```go
pipe.RegisterParser(parser.NewNeo4jParser())
```

`executeNoSQLRead()` 的 switch 加 case：
```go
case "neo4j":
    return executeNeo4jRead(ctx, userID, proj, ds, sqlToExecute, classifyResult)
```

以及新增 `executeNeo4jRead()` 函数。

#### 4. `features/ticket/service.go` — 写执行

```go
case "neo4j":
    // Neo4j 写操作执行
```

---

### 加一个新大类（如 MQ 族 Kafka）

**比 NoSQL 多一步：**

1. `pkg/pipeline/instruction.go` 的 `TypeGroup` 加 `GroupMQ`（如有这个 group 则跳过）
2. `pkg/pipeline/instruction.go` 的 `DatasourceTypes` 加 `"kafka"` → MQ
3. `pkg/driver/kafka.go` — 连接辅助函数
4. `pkg/pipeline/parser/kafka.go` — Parser
5. `features/exec/service.go` `Execute()` 加 `case "mq":` + `executeMQRead()`
6. `features/ticket/service.go` `ExecuteTicket()` 加 `case "mq":`

---

## 现有 Parser 扩展

| Parser | 位置 | 扩展方式 |
|--------|------|---------|
| SQL | `pkg/pipeline/parser/sql.go` | 往 `sqlKeywordPatterns` 表加行即可 |
| Redis | `pkg/pipeline/parser/redis.go` | 往 `redisCommands` 表加行即可 |
| MongoDB | `pkg/pipeline/parser/mongo.go` | 往 `mongoCommands` 表加行即可 |
| ES | `pkg/pipeline/parser/es.go` | 往 `esPatterns` 表加行即可 |

---

## 无需改动

| 层 | 原因 |
|----|------|
| 工单审批（ticket） | 按 TypeGroup 分发，写操作执行只加一个 case |
| 审计（audit） | 类型无关 |
| Webhook（webhook） | 类型无关 |
| 项目权限（project） | 类型无关 |
| 规则引擎（dsrule） | 规则通过 `type_group` + `type_scope` 双字段作用，加新类型自动匹配 |
| 连接池监控（poolstats） | 通用 |
