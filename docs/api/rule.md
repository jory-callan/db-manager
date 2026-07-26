# 数据源分类规则管理

> 规则引擎对指令解析器输出的分类结果进行二次判定，可覆盖 parser 的初步分类。
> 规则通过 `type_group`（大类）+ `type_scope`（小类）双字段匹配数据源类型。

## 接口一览

| 方法 | 路径 | 权限码 | 说明 |
|------|------|--------|------|
| GET | `/api/v1/datasource-rules` | `db:rule:view` | 规则列表 |
| POST | `/api/v1/datasource-rules` | `db:rule:create` | 创建规则 |
| PUT | `/api/v1/datasource-rules/:id` | `db:rule:edit` | 编辑规则 |
| DELETE | `/api/v1/datasource-rules/:id` | `db:rule:manage` | 删除规则 |
| POST | `/api/v1/datasource-rules/batch-delete` | `db:rule:manage` | 批量删除规则 |

## 规则匹配机制

### 双字段适用范围

| type_group | type_scope | 匹配效果 |
|-----------|-----------|---------|
| `""` | `""` | 所有数据源 |
| `"sql"` | `""` | 所有 SQL 数据源（mysql/pg/tidb/oceanbase/mariadb） |
| `""` | `"mysql"` | 仅 MySQL |
| `"sql"` | `"mysql"` | 仅 MySQL（双重限定）|

### 匹配流程

1. 按 `priority` 升序排序，数字小优先
2. 规则匹配：`(type_group = dsGroup OR type_group = '') AND (type_scope = dsType OR type_scope = '')`
3. 按 priority 顺序，第一条正则命中的规则即生效
4. 规则可覆盖 Parser 的初步分类（如 Parser 判为 write，规则命中 dangerous 则改为 dangerous）

### 内置 seed 规则

系统默认初始化以下规则，type_group="sql" AND type_scope="" 自动覆盖所有 SQL 类型数据源：

| name | type_group | type_scope | pattern | category | priority |
|------|-----------|-----------|---------|----------|----------|
| SELECT | sql | | `^\s*SELECT\b` | read | 0 |
| INSERT | sql | | `^\s*INSERT\b` | write | 0 |
| UPDATE | sql | | `^\s*UPDATE\b` | write | 0 |
| DELETE | sql | | `^\s*DELETE\b` | write | 0 |
| CREATE | sql | | `^\s*CREATE\b` | dangerous | 0 |
| ALTER | sql | | `^\s*ALTER\b` | dangerous | 0 |
| DROP | sql | | `^\s*DROP\b` | dangerous | 0 |
| TRUNCATE | sql | | `^\s*TRUNCATE\b` | dangerous | 0 |

## POST /api/v1/datasource-rules

请求体：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 规则名称 |
| type_group | string | 否 | 类型分组：空=全部 / sql / nosql / search / mq |
| type_scope | string | 否 | 具体类型：空=全部 / mysql / redis / es（留空则作用于整个分组）|
| category | string | 是 | 分类：read / write / dangerous |
| pattern | string | 是 | 正则表达式 |
| enabled | bool | 否 | 默认 true |
| priority | int | 否 | 排序优先级，数字越小越优先，默认 0 |
