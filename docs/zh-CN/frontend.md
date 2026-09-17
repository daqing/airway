# 前端指南

Airway 的前端管线**全程无需 Node.js**：npm 包的获取、打包、服务都由
`airway` CLI 和 Go 二进制完成。页面由 templ 服务端渲染；交互区域是
**岛屿（islands）**——挂载在页面小块区域上的 Preact 组件。

- English version: [docs/frontend.md](../frontend.md)

## 全景

| 层 | 说明 |
| --- | --- |
| `js.pkg.json` | 依赖清单：`deps`（精确版本，可手改）+ `lock`（全解析树，含 sha512 integrity）。 |
| `app/assets/js/vendor/` | 已安装的包（node_modules 布局，随仓库提交）。 |
| `app/assets/js/islands/` | 你的交互组件；每个文件 default 导出一个岛屿。 |
| `app/assets/js/ui/` | airway-ui 组件库。 |
| `app/assets/css/airway.css` | 设计令牌 + 组件样式（亮/暗双主题）。 |
| `app/assets/dist/` | `js:build` 产物（随仓库提交），经 go:embed 打进二进制。 |

## 命令

```bash
airway js:add preact                        # 添加依赖（latest，精确锁定）
airway js:add @tanstack/react-table@9.2.4   # 指定版本添加
airway js:install                           # 按 js.pkg.json 安装（幂等）
airway js:build                             # 打包 app/assets/js -> app/assets/dist
airway generate island chart                # 生成岛屿组件骨架
airway generate scaffold post title:string  # 完整 CRUD：模型、迁移、API、页面、岛屿
```

镜像源：`AIRWAY_JS_REGISTRY`（默认 `https://registry.npmjs.org`，可切
`https://registry.npmmirror.com`）。

## 开发流程

`AIRWAY_ENV=local` 下，`airway server` **在内存中**构建 bundle：
`/assets/*` 始终返回最新构建；保存 `app/assets/js` 下任何
`.ts/.tsx/.css` 文件（vendor 除外）会触发增量重建并经 WebSocket
livereload 整页刷新。没有 HMR 服务器、没有额外 watcher 进程——全部
在 server 二进制内。

生产环境提交的 `dist/` 由 `go:embed` 打进单二进制，部署形态不变。

## 岛屿

在任意 templ 视图中嵌入：

```go
@assets.Island("counter", map[string]any{"start": 3})
```

渲染为：

```html
<div data-island="counter" data-island-id="1"></div>
<script type="application/json" id="island-data-1">{"start":3}</script>
```

运行时扫描 `[data-island]` 节点、解析相邻 JSON 为 props、挂载同名组件。
规则：

- 岛屿名 = `islands/` 下文件路径去扩展名（`islands/admin/chart.tsx` 即
  `"admin/chart"`），大小写敏感。
- 每个岛屿文件 **default 导出** 组件。
- 新增文件无需注册，下次构建自动生效。
- 禁用 JS 时页面优雅降级：挂载点为空、props 保持惰性 JSON。

源码 `import "react"`（构建时别名到 preact/compat），因此遵循 React
语义——与 `register()` 配合的自定义输入必须 `forwardRef`。

## airway-ui

岛屿所用的组件库（任意运行中的服务器 `/ui` 可看全部实况）：

| 类别 | 组件 |
| --- | --- |
| 基础 | Button、Input、Textarea、Select、Checkbox、Radio、Field、Spinner、EmptyState |
| 表单 | Form（react-hook-form 集成） |
| 数据 | DataTable（TanStack Table：排序、分页、行选择、加载/空态） |
| 反馈 | Modal（焦点圈、Esc）、Toast（`useToast`）、Tabs、Pagination |
| 请求 | `apiFetch`、`useApiQuery`、`ApiError`、`setApiErrorHandler` |

请求层对齐 `lib/render` 信封 `{"code":0,"data":…,"message":""}`
（code 0 成功）。`setApiErrorHandler(fn)` 是业务错误全局钩子
（如 401 跳转）。

## 教程：从 scaffold 到自定义岛屿

1. 生成 CRUD 资源并启动：

   ```bash
   airway generate scaffold post title:string
   go generate ./...
   airway js:build
   airway db:migrate
   airway server   # http://127.0.0.1:1900/posts
   ```

2. 生成自定义岛屿并嵌入：

   ```bash
   airway generate island chart
   ```

   ```go
   // 在 .templ 视图中
   @assets.Island("chart", map[string]any{"points": []int{4, 8, 15}})
   ```

3. 构建（local 下保存文件即可）并刷新：

   ```bash
   airway js:build
   ```

编辑 `app/assets/js/islands/chart.tsx`，用 airway-ui 组件拼装——
这就是完整的开发循环。
