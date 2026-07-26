# DataTable

通用数据表格组件，基于 `@tanstack/react-table` + shadcn/ui Table。

## 快速开始

```tsx
import { DataTable } from '@/components/data-table'
import type { ColumnDef } from '@tanstack/react-table'

interface User { id: number; name: string; email: string }

const columns: ColumnDef<User>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'name', header: '姓名' },
  { accessorKey: 'email', header: '邮箱' },
]

function Page() {
  const { data, isFetching } = useQuery(...)

  return (
    <DataTable
      columns={columns}
      data={data?.list ?? []}
      loading={isFetching}
      getRowId={(row) => String(row.id)}
    />
  )
}
```

---

## Props

### 基础

| Prop | 类型 | 默认 | 说明 |
|---|---|---|---|
| `columns` | `ColumnDef<TData>[]` | 必填 | TanStack Table 列定义 |
| `data` | `TData[]` | 必填 | 表格数据（通常是 API 返回的 list） |
| `loading` | `boolean` | `false` | 加载状态，loading + 空数据显示骨架 |
| `emptyText` | `string` | `'暂无数据'` | 空数据时的提示文字 |
| `className` | `string` | - | 外层容器 class |

### 排序

| Prop | 类型 | 默认 | 说明 |
|---|---|---|---|
| `enableSorting` | `boolean` | `true` | 列头点击排序（asc / desc / none） |

### 行选择

| Prop | 类型 | 默认 | 说明 |
|---|---|---|---|
| `enableRowSelection` | `boolean` | - | 启用左侧勾选列 |
| `getRowId` | `(row, index) => string` | - | 行唯一 ID，行选择必传 |

### 分页

| Prop | 类型 | 说明 |
|---|---|---|
| `pagination` | `DataTablePagination` | 分页配置 |

`DataTablePagination`:

```ts
{
  page: number
  pageSize: number
  total?: number         // 总数（needCount=true 时必传）
  needCount?: boolean    // 是否需要显示总数
  pageSizeOptions?: number[]  // 每页条数选项，默认 [10,20,50,100,200,500,1000,2000]
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
}
```

### 工具栏

| Prop | 类型 | 说明 |
|---|---|---|
| `toolbar` | `DataTableToolbar<TData>` | 工具栏配置 |

```ts
{
  searchPlaceholder?: string          // 全局搜索占位文字
  createLabel?: string                // 创建按钮文字
  deleteLabel?: string                // 批量删除按钮文字
  refreshLabel?: string               // 刷新按钮文字
  refreshIntervalOptions?: number[]   // 自动刷新间隔选项（秒），如 [10,30,60]
  extraActions?: { label, icon, variant, onClick }[]  // 自定义操作
  onCreate?: () => void
  onRefresh?: () => void
  onBatchDelete?: (rows: TData[]) => void  // 选中行批量删除
}
```

### 列级筛选（受控模式）

DataTable 只负责渲染筛选控件和通过 `onFiltersChange` 吐回值，不碰数据。
父组件收到新值后自行触发 API 请求并重置页码。

| Prop | 类型 | 说明 |
|---|---|---|
| `filters` | `DataTableFilter[]` | 筛选器定义 |
| `filterValues` | `Record<string, string>` | 当前值（受控） |
| `onFiltersChange` | `(values) => void` | 值变化回调 |

`DataTableFilter`:

```ts
{
  id: string           // 筛选器标识，同时也是 API query 参数名
  label: string        // 展示标签
  type: 'select' | 'input'
  options?: { value, label }[]   // select 类型的选项
  placeholder?: string
  defaultValue?: string          // 默认值，如 '_all'
}
```

#### 用法示例

```tsx
const [filterValues, setFilterValues] = useState<Record<string, string>>({})

const query = useQuery({
  queryKey: ['list', page, pageSize, filterValues],
  queryFn: () => api.list({ page, pageSize, ...filterValues }),
})

<DataTable
  columns={columns}
  data={data?.list ?? []}
  loading={isFetching}

  // 分页（受控）
  pagination={{
    page, pageSize, total: data?.total, needCount: true,
    onPageChange: setPage,
    onPageSizeChange: (size) => { setPageSize(size); setPage(1) },
  }}

  // 筛选（受控）
  filterValues={filterValues}
  onFiltersChange={(values) => { setFilterValues(values); setPage(1) }}
  filters={[
    {
      id: 'status',
      label: '状态',
      type: 'select',
      options: [
        { value: '_all', label: '全部' },
        { value: '=active', label: '活跃' },
        { value: '=disabled', label: '禁用' },
      ],
    },
    {
      id: 'name__contains',
      label: '名称',
      type: 'input',
      placeholder: '搜索名称...',
    },
  ]}
/>
```

筛选值变化路径：

```
用户操作 → onFiltersChange({ status: '=active' })
  → parent setFilterValues → parent setPage(1)
    → queryKey 变化 → API 重新请求
      → 新 data 和 total 传回 DataTable
```

> `filterValues` 中的值是纯字符串格式，后端按约定的 query 协议解析。
> 当前约定的操作符前缀：
> - `=value` — 精确匹配
> - `=～*val*` — LIKE（`*` 转 `%`）
> - `=[a,b]` — IN
> - `=gte:val` / `=lte:val` / `=gt:val` / `=lt:val` — 比较
> - `=between:a,b` — BETWEEN
> - `_all` 或不传 — 不过滤

---

## 布局说明

- 紧凑间隙设计（`gap-2` / `gap-3`），适合信息密集型页面
- 工具栏和筛选器在同一层级，筛选器渲染在工具栏下方
- 分页控件含页码跳转、每页条数选择
- 全局搜索（工具栏内的搜索框）在 DataTable 内部维护状态，不影响父组件

## 测试

开发模式访问 `/test/data-table` 查看各种用法示例，包括列级筛选 Tab。
