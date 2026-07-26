# web/CONVENTIONS.md — 前端技术约定

> 修改 web/ 代码前必读。
> 系统架构与项目级约定见 ../CONVENTIONS.md。

## 自解释 / 无状态原则

本项目必须做到仓库内自解释：任何无历史记忆、无外部 skill 的开发者或 AI Agent，只阅读项目文档就能理解技术栈、目录职责、开发命令、路由、API、组件放置规则和验证方式。

不要把关键约定只放在聊天记录、个人记忆或外部工具配置里；新约定稳定后必须同步写回项目文档。

## 快速概览

| 类别 | 约定 |
|---|---|
| 包管理 | 必须使用 pnpm，不用 npm |
| UI 组件 | shadcn/ui，用 CLI 安装（`pnpm dlx shadcn@latest add`） |
| 业务组件 | `src/components/common/` |
| 路由 | react-router-dom v7 `createBrowserRouter`，集中 `routes.tsx` |
| API 客户端 | ofetch，基础路径 `/api/v1` |
| 服务端状态 | @tanstack/react-query，默认不自动重试 |
| 客户端状态 | zustand |
| 国际化 | 当前不启用，未来按需恢复 |
| 图标 | lucide-react / @tabler/icons-react，禁止手写 SVG |
| 样式 | TailwindCSS 4，不写自定义 CSS |
| 表格 | @tanstack/react-table + shadcn/ui Table，优先用 `data-table.tsx` |
| 文件名 | TS/TSX 用 kebab-case，组件名用 PascalCase |
| 路由路径 | kebab-case |
| DataTable | 默认 `layout="fill"`，分页查询用 `placeholderData: keepPreviousData` |
| 分页 | 前端传 `page` / `page_size` / `need_count`；后端返回 `list` / `total` / `page` / `page_size` |
| page_size | 最大 2000 |

## 大前提

- Node 包管理默认使用 pnpm：安装依赖用 `pnpm install` / `pnpm add`，不使用 npm
- shadcn/ui 组件使用官方 CLI 安装，不手写复制组件源码：`pnpm dlx shadcn@latest add <component>`
- 如果参考文档写的是 `npx shadcn@latest add <component>`，本项目等价改用 `pnpm dlx shadcn@latest add <component>`
- 不需要在 package.json 强制声明 pnpm；这是项目协作推荐，但开发和 AI Agent 操作必须优先采用 pnpm
- `src/components/ui/` 是 shadcn/ui 默认组件目录，固定给 shadcn CLI 使用；业务自定义组件放在 `src/components/common/` 或业务模块目录
- 一般不阅读、不审查、不手改 `src/components/ui/` 下的默认组件实现；只需要知道组件导出和用法，避免浪费上下文
- 禁止手写 SVG：不要手写 inline `<svg>`、`<path>` 或 SVG 文件；图标优先用 lucide-react、@tabler/icons-react 或组件库，示意图优先用 Mermaid/Excalidraw，图片资产使用外部生成或已审核文件

## 技术栈

| 层 | 选型 |
|---|---|
| 框架 | React 19 + TypeScript |
| 构建 | Vite |
| 路由 | react-router-dom v7 `createBrowserRouter` |
| 服务端状态 | @tanstack/react-query |
| HTTP 客户端 | ofetch |
| 表格 | @tanstack/react-table + shadcn/ui Table |
| 客户端状态 | zustand |
| 样式 | TailwindCSS 4 + shadcn/ui |
| 图标 | lucide-react + @tabler/icons-react |

## 目录结构

```
src/
├── components/
│   ├── ui/              # shadcn/ui 组件，默认目录，不作为业务组件目录
│   └── common/          # 项目通用组件，如 DataTable
├── config/              # 前端运行配置，如标题、版本、测试路由开关
├── hooks/               # 通用 hooks
├── lib/
│   ├── api/             # API hooks，按领域分文件
│   ├── api-client.ts    # ofetch 实例
│   ├── query-provider.tsx
│   └── utils.ts
├── pages/               # 路由级页面组件
├── stores/              # zustand 客户端状态
├── routes.tsx           # 集中路由定义
├── App.tsx              # 布局组件
└── main.tsx             # 入口
```

## 路由约定

保持 `routes.tsx` 清爽集中，不主动引入 loader/action 等复杂机制。

- 需要登录的页面放在 App 布局下
- 公开页面放在根路径
- 路由路径使用 kebab-case
- 鉴权跳转函数统一放 `src/lib/auth-redirect.ts`
- 未登录访问受保护页面跳到 `/login?redirect=当前路径`
- 已登录访问 `/login` 自动回首页
- `/login` 使用 `GuestGuard` 验证 token，验证期间显示深色背景 loading
- 首屏主题 class 由 `index.html` 内联脚本提前设置，避免暗黑模式白闪

## API 调用约定

所有 API 调用统一通过 `src/lib/api-client.ts` 的 ofetch 实例：

- 基础路径：`/api/v1`
- 成功：HTTP 200，响应 `code` 为 `0`
- 失败：HTTP 使用真实错误状态码，响应 `code` 和 HTTP 状态码一致
- 响应格式：`{ code: number, msg?: string, data: any }`
- `apiClient` 自动解包 `data`
- TanStack Query 默认不自动重试
- 401 自动清理 token 并跳转 `/login`

## 登录态和用户信息

- 登录态和当前用户信息统一放 `src/stores/app-store.ts`
- 登录成功用 `setAuth(token, user)` 保存
- 退出用 `clearAuth()`
- 鉴权等待态统一用 `AuthLoadingScreen`，背景使用 `bg-background text-foreground`

## 组件规范

- 页面组件放 `src/pages/`，按路由一一对应
- 通用业务组件放 `src/components/common/`
- shadcn/ui 组件放 `src/components/ui/`
- 使用 TailwindCSS 类名，不写自定义 CSS
- TS/TSX 文件名使用 kebab-case，组件名使用 PascalCase

## 弹窗 / 侧边栏规范

| 用途 | 组件 | 理由 |
|------|------|------|
| 新增 / 编辑表单（3+ 字段） | Sheet | 全高视图，高度固定不跳动，适合多 tab 或复杂配置 |
| 查看详情（多 section） | Sheet | 右侧滑入，不覆盖列表上下文 |
| 成员管理、关联配置等面板 | Sheet | 与主表单风格保持一致 |
| 确认类操作（删除、状态变更） | Dialog / AlertDialog | 轻量中断式交互，无需大面板 |
| 单字段表单（输入名称） | Dialog | 一两行字段，弹窗够轻 |
| 提示 / 通知 | sonner toast | 非阻塞，统一 toast 方案 |

**原则：** CRUD 表单优先使用 Sheet；Dialog 回归其本来的职责——确认和轻量交互。不混用，不互相替代。

## 页面目录规范

适用于 `pages/<route>/` 下的每个页面：

- **page.tsx** — 入口文件，只负责组装子组件和全局逻辑（URL 参数、API 调用），不做视图渲染
- **store.ts** — 页面级 zustand store（不放到 `src/stores/`），管理表单/弹窗/选择等跨组件状态
- **columns.tsx** — 列定义（DataTable 使用），单独文件
- 一个界面拆分多个子文件，**单文件不超 300 行**
- 数据和展示分离：API/mutation 在 page.tsx，视图组件抽离为子文件
- 参数优先通过路由 URL 传递（`useSearchParams`），刷新/分享保留状态
- ID 列默认不展示（隐藏主键）
- 表格标配批量删除：`enableRowSelection` + `onBatchDelete`
- 后端字段名对齐：避免 `description`/`remark` 混淆，以后端 DTO 为准

## 布局风格

- **信息密集型布局**，间隙小
- 表格上方不设分隔 Card，标题紧凑（`text-lg`），操作栏和表格连成一体
- 行高、间距使用 `gap-2`/`gap-3` 级别，不滥用 `gap-6`/`py-4` 等大间距

## 表格规范

- 分页表格优先使用 `src/components/common/data-table.tsx`
- 使用 TanStack Table 管状态/逻辑，shadcn/ui Table 管视觉
- 后端分页：`page`、`page_size`、`need_count`；返回：`list`、`total`、`page`、`page_size`
- `page_size` 最大 2000
- 分页查询使用 `placeholderData: keepPreviousData`
- DataTable 默认 `layout="fill"`，小表格用 `layout="auto"`
- 每个路由传独立 `storageKey`，列显示配置按页面持久化
