# Frontend Plan — esbuild-in-CLI + Preact Islands

实施计划：为 Airway 引入组件化的前端交互层。决策背景见
[README.md](README.md) 的 *Frontend strategy* 章节（templ 骨架 + Preact
岛屿 + esbuild 以 Go 库内嵌 CLI，全程无 Node 工具链）。

本文件是开发路线图：每个 Phase 可独立交付、独立验收，完成后勾选任务。
顺序即建议的实施顺序，Phase 0 未通过前不进入 Phase 1。

## 不变量（每个 Phase 都不得破坏）

- **无 Node**：从 `airway new` 到部署，任何环节不要求安装 Node.js/npm；
  CI 用纯 Go 镜像（如 `golang:alpine`）验证。
- **单二进制部署**：前端产物 `go:embed` 进二进制，部署形态不变。
- **向后兼容**：现有 templ 视图、CLI 命令、项目布局不 breaking；
  岛屿是增量能力，不用岛屿的项目零感知。
- **生成物提交**：沿用 `*_templ.go` 的惯例，`app/assets/dist/` 构建产物
  提交进仓库，clone 即可 `go build`。

## 目标形态（完成后）

```
app/assets/
  js/
    app.tsx            # 入口：扫描 data-island 并挂载
    islands/           # 页面交互岛屿（项目代码 + generate island 产物）
    registry.gen.ts    # 构建期生成的岛屿注册表
    ui/                # airway-ui 组件库（框架仓库内维护）
    vendor/            # airway js:install 拉取的第三方包（node_modules 兼容布局）
  css/
    airway.css         # design tokens + 组件样式
  dist/                # airway js:build 产物（提交进仓库）
    app.js
    manifest.json      # 入口 -> 文件、构建 hash
```

页面数据流：templ 渲染 HTML 骨架 → `data-island="OrderList"` 挂载点 +
`templ.JSONScript` 序列化初始数据 → `app.tsx` 扫描挂载点、反序列化
props、渲染 Preact 组件 → 岛屿通过 fetch 调 JSON API（`lib/render` 约定）。

## 前端库支持矩阵（已定调）

TanStack 系是「纯逻辑 core + 薄框架 adapter」架构：逻辑型库可以落地
（必要时直达 core 自写 adapter），工具链级产品（Router / Start）不可
用。岛屿架构里路由本来就在服务端（一个 templ 页面即一个路由），不
需要客户端 SPA router；哪天一个应用真的需要 Router 级前端，即触发
「拆出 Vue 前端」的逃生舱，不在本方案内硬撑。

| 库 | 结论 | 说明 |
| --- | --- | --- |
| `@tanstack/preact-query` | ✅ 官方支持 | 数据层选型；官方 Preact 适配，不走 compat 别名 |
| `@tanstack/react-table` | ✅ 可用 | v9 移除了 v8 hooks API，走 `./legacy` 出口的 v8 API（Phase 0 实测排序/分页通过）；兼容缺口兜底：对 `@tanstack/table-core` 自写 adapter |
| `@tanstack/react-virtual` | ✅ 大概率可用 | via compat；Phase 0 逐项验证 |
| `react-hook-form` | ✅ 官方支持 Preact | 表单选型，优先于 TanStack Form |
| `@tanstack/react-form` | ⚠️ 能跑但非官方 | 不选；若引入需接受社区自扛定位 |
| `@tanstack/router` | ❌ 不支持 | 依赖 React 并发语义与 Vite 插件（JS 编译器，esbuild Go API 跑不了）；岛屿架构也用不到 |
| `@tanstack/start` | ❌ 不支持 | 绑定 Vite + Nitro + RSC 工具链 |

airway-ui 依赖定调：数据层 `@tanstack/preact-query`，表格
`@tanstack/react-table`（compat），表单 `react-hook-form`。

## 总览

| Phase | 主题 | 一句话目标 |
| --- | --- | --- |
| 0 | 技术验证 | 用 spike 证明四个核心假设成立，消除最大风险 |
| 1 | JS 依赖管理 | `airway js:add` / `js:install`：Go 实现的 npm 包获取与锁定 |
| 2 | 构建管线 | `airway js:build` + dev 内存构建 + embed + livereload |
| 3 | 岛屿运行时 | templ 与 Preact 的挂载协议、helper、注册表 |
| 4 | airway-ui | 组件库底座与首批组件（Form/Table 等 CRUD 全套） |
| 5 | 脚手架整合 | generate 系列与 airway new 模板产出前端代码 |
| 6 | 文档与发布 | 文档、README 状态更新、CI、版本发布 |

---

## Phase 0 — 技术验证（spike，代码可丢弃）

**目标**：以最小代价验证整个方案的四个技术假设，任何一条不成立都要在
此阶段暴露并调整设计，而不是在 Phase 2 中途返工。

任务：

- [x] spike 隔离在 `spike/frontend/` 目录（在 feat/front-end 分支上，
  未单独建分支），`go.mod` 引入 `github.com/evanw/esbuild`，用
  `pkg/api` 构建（`Bundle: true`、`Format: ESModule`、
  `JSX: automatic` + `jsxImportSource: "preact"`）编译 Preact TSX，
  产物可运行。
- [x] 验证 `preact/compat` 别名：esbuild `Alias` 把 `react` /
  `react-dom` 指到 `preact/compat` 后，按「前端库支持矩阵」逐项跑
  demo，确认能构建并在浏览器正常工作：
  - `@tanstack/preact-query`（官方包，不走 compat）：请求 + 缓存；
  - `@tanstack/react-table`（compat）：排序 + 分页；
  - `@tanstack/react-virtual`（compat）：长列表虚拟滚动；
  - `react-hook-form`（compat）：受控表单 + 校验。
- [x] 验证 Watch/Rebuild：v0.28 已移除 `Watch` 回调，实际形态为
  `Context` + `Watch` 后台增量重建、请求时 `Rebuild()` 取产物
  （goroutine 中运行，进程不退出）。
- [x] 验证内存 serve：gin middleware 直接返回 `OutputFiles` 字节，
  Content-Type / ETag 正确。
- [x] 测量二进制体积增量（实测 esbuild 纯增量约 +3.8MB，优于预估的
  8~10MB），结论见下。
- [x] 把结论（含每条假设的通过/失败证据）追加到本文件。

**验收**：一个临时页面上，TSX 写的「表格 + 表单」岛屿跑通；全流程
未安装 Node。

**风险**：preact/compat 对个别库的兼容缺口。若 TanStack 某库不可用，
降级策略是替换为可在 Preact 原生 API 下工作的等价库，不影响架构。

### Phase 0 结论（2026-09-17 执行，四项假设全部通过）

spike 位于 `spike/frontend/`（作为 Phase 1–3 的参考实现保留；复现：
`go run . --fetch-deps && go run . [--watch]`，依赖经 npmmirror 拉取，
全程无 Node）：

- **esbuild Go API（v0.28.2）内存构建 Preact TSX**：通过。`JSX
  automatic` + `jsxImportSource: "preact"` + `react`→`preact/compat`
  别名，bundle 240KB（minified，四个库 + demo 全量）。
- **preact/compat 兼容性**：四个库在浏览器实测全部通过——
  `@tanstack/preact-query@5.103.1`（官方适配：两个同 key 组件只发
  一次请求 `requestCount=1`，invalidate 后同步变 2，证明缓存共享）；
  `@tanstack/react-table@9.2.4`（**v9 已移除 v8 hooks API**，走
  `./legacy` 出口保留的 v8 API：排序、分页实测正确）；
  `@tanstack/react-virtual@3.14.13`（scrollTop=5000 后窗口精确移至
  row 170–195，10000 行整页仅 34 个 div）；
  `react-hook-form@7.88.0`（required/pattern 校验、handleSubmit、
  register spread 全链路正常）。
- **Watch/Rebuild**：通过，但 **v0.28 无 OnRebuild 回调**——`Watch()`
  只负责后台轮询增量重建，产物需显式 `Rebuild()` 获取。dev middleware
  定型为：启动 Watch + 请求时 `Rebuild()`（源码未变时 ~30ms no-op；
  变更后增量重建 28ms，下一个请求即拿到新 ETag，浏览器 reload 后
  页面正常）。
- **内存 serve**：通过。ETag / `If-None-Match` 304 / Content-Type 全
  对；`.map` 不在 macOS 系统 mime 表，需特判为 `application/json`。
- **体积**：esbuild 纯增量实测 **+3.8MB**（blank-import 对照 5.66MB
  vs 1.83MB，strip 后 3.8MB），优于预估；主程序不 import esbuild，
  二进制完全不受影响（40.2MB 不变）。未来 `js:build` 进 CLI 后仅
  构建/CLI 二进制变大。
- **依赖获取（Phase 1 原型顺带验证）**：纯 Go 下载器（packument 解析
  + tarball 解包 + 递归 dependencies）拉齐 11 个包；npmmirror 可达性
  远好于 registry.npmjs.org（本机 0.35s vs 超时），`--registry` 参数
  化保留。

## Phase 1 — 前端依赖管理（无 Node 的包获取）

**目标**：解决「npm 包从哪来」：不引入 Node，由 airway CLI 直接与
npm registry（HTTP API + tarball）交互，安装到项目本地。

任务：

- [x] 定义 `js.pkg.json`：`deps` 为直接依赖的**精确版本**（lock 语义）；
  `lock` 为全解析树（含传递依赖 + sha512 integrity）。传递依赖来自 npm
  的 range 语法，绕不开——已实现最小 range 解析（精确 / `^` / `~` /
  比较符 / x-range / `||` / AND，prerelease 门控），见 `lib/jspkg/semver.go`。
- [x] 实现 `airway js:add <pkg>[@<version>]`：落地为 `lib/jspkg` 包（可
  复用）+ `cmd/cli_js_cmd.go` 命令入口（跟随 cmd/ 平铺惯例；注意 Go
  文件不能命名为 `*_js.go`——`_js` 后缀会被解析为 GOOS=js 隐式构建
  约束而静默排除）。从 registry 取元数据、下载 `dist.tarball`、解包
  到 `app/assets/js/vendor/`（node_modules 兼容布局），校验 SRI
  integrity 后写入 `js.pkg.json`。
- [x] `airway js:install`：按 lock 精确安装；本地版本一致跳过（实测
  幂等重装零下载）；lock 缺失/过期时按 deps 重新解析。
- [x] registry 可配置：`AIRWAY_JS_REGISTRY`（别名 `JS_REGISTRY`），默认
  `https://registry.npmjs.org`。
- [x] 基础依赖预置：`airway new` 的模板含 `js.pkg.json`（预置 preact、
  `@tanstack/preact-query`、`react-hook-form` 及完整 lock）；airway-ui
  基础包推迟到 Phase 4 诞生时加入。Next steps 提示 `airway js:install`。
- [x] 单测（`lib/jspkg/jspkg_test.go`，全部离线）：semver 表驱动、
  spec 解析、tarball 解包、SRI 校验、镜像 URL、httptest fixture 的
  依赖树解析 / 幂等安装 / scoped 包。
- [x] 决策落地：`vendor/` 提交进仓库——模板 `.gitignore` 不排除
  `app/assets/js/vendor/`。

**验收**：`airway js:add @tanstack/react-table@<版本>` 后，TS 里
`import ... from "@tanstack/react-table"` 能被 js:build 成功解析。

### Phase 1 结论（2026-09-17 执行，验收通过）

真实环境（npmmirror，全程无 Node）端到端验证：`js:add preact` +
`js:add @tanstack/react-table@9.2.4` 递归锁定 6 包（react-store /
store / table-core / use-sync-external-store 含在内），二次 add 复用
已装包；esbuild 以 vendor 为 NodePaths、react→preact/compat 别名成功
bundle `@tanstack/react-table` + `./legacy` 出口 + preact（311KB，
零错误零警告）。`airway new` 脚手架验证含预置 `js.pkg.json`
（3 deps + 4 lock 条目）。`go build` / `go vet` / `go test ./...` 全绿
（26 包）。

与计划的偏差（均为实施中发现的设计修正）：

- 位置：`lib/jspkg` 库 + `cmd/cli_js_cmd.go`，不是 `cmd/jspkg` 子包。
- lock 携带 npm SRI（sha512）integrity，跨镜像安装可验证内容一致。
- 扁平 vendor 布局下同包多 range 冲突策略：选满足全部 ranges 的最高
  版本，无共同版本时报错（npm 会嵌套 node_modules，v1 不做）。
- 预置依赖暂不含 airway-ui（Phase 4）。

## Phase 2 — 构建管线与开发流程

**目标**：`airway js:build` 出产物、dev 时内存构建、livereload、
`go:embed` 单二进制，前端开发闭环成立。

任务：

- [x] `airway js:build`：落地为 `lib/jsbuild` 包 + `cmd/cli_js_cmd.go` 命令。
  入口 `app/assets/js/app.tsx`，`vendor/` 为 node 解析路径，统一注入
  `react`→`preact/compat` 别名；minify、sourcemap、`Target: ES2020`。
- [x] 固定文件名（`app.js`）+ `manifest.json`（entry、hash、files）；
  缓存策略用查询串 `?v=<hash>`（`app/assets.EntryPath()`），裸路径走
  ETag 协商。构建确定性已验证：同输入两次构建 hash 相同。
- [x] 产物写入 `app/assets/dist/` 并提交；`app/assets` 包（仓库无
  internal/ 惯例，未按计划用 `internal/assets`）以 `//go:embed all:dist`
  暴露 `http.Handler`，`config/routes.go` 挂 `/assets/*`；`URL_PREFIX`
  由既有 prefixHandler 剥离后透传（实机验证 /airway/assets/… 200，裸根
  /health 仍 200）。
- [x] dev 内存构建：`AIRWAY_ENV=local` 时 `main.go` 调
  `jsbuild.StartDefault`，`/assets/*` 由内存产物应答（ETag 协商 304）；
  esbuild `Watch` 常驻后台。
- [x] livereload：自研 mtime 轮询（300ms，跳过 vendor/）触发显式
  `Rebuild()` + `websocket.Broadcast`（`app/websocket` 新增导出函数）
  广播 `{"type":"js-rebuild"}`；`base.templ` 在 local 下注入
  `/assets/livereload.js`（ws 路径经 `data-ws` 适配 URL_PREFIX）。
- [x] `Procfile.dev` 无需改动（dev 构建在 server 进程内，随 air 重启）；
  `justfile` 新增 `build` recipe（js:build 前置 + go build）。
- [x] 单测（全部离线）：`lib/jsbuild`（manifest 与确定性、dev handler
  200/304/404、源码变更→重建→广播、scanMtimes 忽略 vendor）、
  `app/assets`（embed handler、ETag/immutable 双模式、manifest/
  EntryPath、路径穿越 404）。

**验收**：改一个 `.tsx` 保存，浏览器约 1 秒内自动刷新；`go build` 出的
二进制拷到无源码机器可正常 serve `/assets/app.js`。

### Phase 2 结论（2026-09-17 执行，验收通过）

- **改 .tsx → 浏览器自动刷新**：实机验证——页面植入 `window` 标记后
  修改 `app.tsx`，watcher 触发增量重建（日志两次 rebuilt），WebSocket
  广播后页面刷新、标记消失，总延迟约 1s。
- **单二进制 serve**：纯二进制拷至空目录 + `URL_PREFIX=/airway` 运行，
  `/airway/assets/app.js`（ETag 协商）、`/airway/assets/manifest.json`、
  裸根 `/health` 均 200；`?v=<hash>` 响应 `immutable` 一年缓存，裸路径
  `no-cache` + ETag。
- **构建确定性**：空入口恢复后重新 `js:build`，hash 与首次完全一致
  （`a80ecb1ad25190cb`），提交的 dist 无噪声 diff。
- `go build` / `go vet` / `go test ./...` 全绿（28 包，含新增
  `lib/jsbuild`、`app/assets`）。

实施要点与坑：

- esbuild `AbsWorkingDir` 必须绝对路径，`Build`/`StartDev` 入口统一
  `filepath.Abs`。
- **watcher 基线竞态**：mtime 初始快照若在 goroutine 内采集，启动与
  首次轮询之间的文件变更会被吞进基线（测试偶发超时的根因）；基线
  改为构造时同步采集。
- embed 要求 dist 至少有一个文件；dist 产物已提交（当前为空入口的
  最小 bundle，Phase 3 island runtime 落地后自然增长）。

## Phase 3 — 岛屿运行时（templ ↔ Preact 协议）

**目标**：定义并实现挂载协议，让 templ 页面以声明式方式嵌入岛屿。

任务：

- [x] 协议定稿：`<div data-island="counter" data-island-id="1"></div>`
  挂载点 + 相邻 `<script type="application/json" id="island-data-1">`
  初始数据（`templ.JSONScript` 生成，实测 `</script>` 转义为
  `\u003c/script\u003e`、name 属性 HTML 转义）。**岛屿名 = islands/
  下文件路径去扩展名，大小写敏感**（验收时踩过：`Counter` ≠
  `counter`）。
- [x] runtime：`app/assets/js/app.tsx` 扫描 `[data-island]`，按注册表取
  组件、解析 props、挂载；未注册时 console 报
  `[island] no island registered for "X"`。修复关键 bug：必须
  `render(h(component, props), el)`——直接调用 `component(props)` 会在
  Preact 组件上下文外执行 hooks（`__H` of undefined）。
- [x] 注册表代码生成：esbuild Go plugin（虚拟模块
  `airway-islands-registry`，OnLoad 每次构建重扫 `islands/`，`_` 前缀
  文件/目录视为 partial 跳过）；新增岛屿文件零手工注册，dev 下经
  mtime watcher 触发重建后自动进 bundle（有测试覆盖）。不生成落盘的
  `registry.gen.ts`（计划写法），虚拟模块每次构建重生成更干净。
- [x] templ helper：`app/assets.Island(name, props)` 输出挂载点 +
  JSONScript；`app/assets.Scripts()` 生产读 embed manifest 注入
  `?v=<hash>`，local 注入 dev 入口。放 `app/assets` 包而非计划写的
  `lib/island`——helper 需读 embed manifest，属应用层。
- [x] `base.templ` 集成：head 尾部 `@assets.Scripts()`，页面无感。
- [x] demo：home 页 hero 区 counter 岛屿，props（start=3、label）由
  action 传入（`home.Index(3)`），作为协议参考实现。
- [x] 单测：`lib/jsbuild`（registry 生成、partial/嵌套跳过、Build 端到
  端含岛屿代码、dev 新增岛屿文件→重建→进 bundle）；`app/assets`
  （挂载点/props 配对、name 与 `</script>` 转义、id 唯一、Scripts 的
  生产/local 两形态）。

**验收**：在一个现有 templ 页面里写 `{{ island.Component("Counter",
map[string]any{"start": 3}) }}`（等价 Go 调用），页面出现可交互组件；
禁用 JS 时页面骨架仍完整。

### Phase 3 结论（2026-09-17 执行，验收通过）

- **dev（内存构建）**：浏览器实机——home 页 counter 岛屿渲染（props
  `{"label":"try an island","start":3}` 从服务端 JSON 传入），点击
  +/− 计数 3→5→2 响应式更新。
- **生产（嵌入 bundle）**：`js:build` + 纯二进制空目录运行，页面加载
  `app.js?v=dbc9e9bbb5069528`，counter 渲染并可交互（3→4）。
- **无 JS 骨架**：curl 生产页面，挂载点为空 div、props 为惰性 JSON，
  页面结构（标题/features/链接）完整——禁用 JS 页面不破。
- 框架仓库自身启用 dogfooding：根目录 `js.pkg.json`（preact@10.29.8）
  + `js:install`，vendor/（1.9MB，131 文件）按 Phase 1 决策提交。
- `go build` / `go vet` / `go test ./...` 全绿（28 包；home 视图测试
  适配 `Index(counterStart)` 签名）。

## Phase 4 — airway-ui 组件库

**目标**：CRUD 场景组件齐备的组件库底座：视觉层自研、逻辑层借
preact/compat 生态。

任务：

- [x] design tokens：`app/assets/css/airway.css`（CSS custom properties：
  色板/圆角/阴影/字体 + `[data-theme="dark"]` 暗色变量），组件零第三方
  CSS 依赖；`aw-` 前缀类名；经入口 import 打包为 `dist/app.css`。
- [x] 组件 API 风格定稿：Preact `class` prop + `cx()` 合并（库类在前、
  调用方在后）；输入类组件全部 `forwardRef`（RHF register 必需）；
  事件透传原生命名；受控属性直传不设内部状态。**关键决策：整个岛屿
  树跑 preact/compat**（源码 `import … from "react"` 经 esbuild 别名到
  compat、`jsxImportSource: "react"`）——纯 preact 模式下函数组件不透
  传 ref，RHF 字段注册失效（实测踩坑）。
- [x] 基础组件：Button（variant/size/loading）、Input、Textarea、
  Select、Checkbox、Radio（均 forwardRef）、Field（label 关联 +
  error/hint）、Spinner、EmptyState。
- [x] Form：RHF 集成（`Form` 接收 useForm 实例，submit 按钮
  isSubmitting 感知）；校验与错误展示由 Field + register 规则组合。
- [x] Table：`DataTable` 基于 `@tanstack/react-table` 的 `./legacy`
  出口（排序、分页、行选择、空态、加载态、`aria-sort`）。
- [x] 反馈组件：Modal（Esc 关闭、Tab 焦点圈、role=dialog、打开时
  focus/关闭时归还）、Toast（provider + useToast、消息队列自动消失）、
  Tabs（role=tablist + 方向键导航）、Pagination。
- [x] 数据层：`ui/data` 的 `apiFetch`/`useApiQuery`/`ApiError`/
  `setApiErrorHandler`。**对齐 lib/render 实际信封
  `{"code":0,"data":…,"message":""}`**（code=0 成功、非 0 业务错误，
  修正了本计划早前 "{ok,data}" 的假设）；query 基于
  `@tanstack/preact-query`，QueryClientProvider/ToastProvider 由
  runtime 统一包裹每个岛屿。
- [x] showcase：`/ui` 路由（`ui_api` + `views/ui` + `ui-showcase`
  岛屿），七个区块全部可交互，含仅用库组件拼装的 CRUD demo（列表 +
  新建/编辑 Modal 表单 + 删除 + Toast）；demo 数据接口
  `/api/v1/ui-demo/items` 走标准 render 信封。
- [x] 无障碍基线：Field label htmlFor（useId 自动 id）、Modal 焦点圈 +
  Esc + aria-modal、Tabs role/键盘、表格 aria-sort、checkbox
  aria-label。

**验收**：仅用 airway-ui 组件即可拼出一个完整的列表 + 新建/编辑 +
删除的 CRUD 页面；showcase 页所有组件可交互。

### Phase 4 结论（2026-09-17 执行，验收通过）

浏览器实机（dev 内存构建）逐项验证：

- **CRUD 全流程**：New post → Modal 表单（RHF 校验拦截空值）→ Create
  → modal 关闭 + "Post created" toast + 新行出现在第 2 页；Edit →
  改题 → "Post updated"；Delete → 行移除 + "Deleted …" toast。仅用
  DataTable/Modal/Form/Field/Input/Button/Toast 拼装，未写一行库外
  UI 代码。
- **Form demo**：空提交出两条 required 错误；填值提交 "Subscribed …"
  toast。
- **Modal**：role="dialog" 唯一、真实键盘 Esc 关闭通过；焦点圈与打开
  聚焦在合成事件下偶发时序问题（真实交互正常，listener 已验证）。
- **DataTable**：点 Title 表头升序排序（aria-sort="ascending"）、
  翻页、行选择 checkbox。
- **数据层**：useApiQuery → `/api/v1/ui-demo/items` → render 信封解析
  → 渲染三条库存数据。
- **生产**：`js:build` + 纯二进制空目录运行，/ui 200、app.js 226KB、
  app.css 7KB 以 text/css 服务、`<link …?v=hash>` 注入验证。
- `go build` / `go vet` / `go test ./...` 全绿（28 包）。

实施修正（已回写本计划）：

1. **岛屿树必须跑 preact/compat**：纯 preact 下 RHF register 的 ref
   不进函数组件（React 语义差异），输入组件 forwardRef + 树切 compat
   后全通。源码统一 `import "react"`。
2. **render 信封实际是 `{code,data,message}`** 而非计划假设的
   `{ok,data}`；`ui/data` 按实际实现。
3. `preact/compat` 不导出 `h`，runtime 挂载改用 `createElement`。
4. 误改 vendor 文件后以「删目录 + js:install 重装」恢复——vendor 必须
   保持 npm 原样。

## Phase 5 — 脚手架整合

**目标**：generate 系列与 `airway new` 模板产出前端代码，scaffold 的
默认产物即是可交互页面。

任务：

- [x] `airway generate island <name>`：生成骨架（`cmd/cli_scaffold.go`），
  经注册表 plugin 自动进 bundle（无需手工注册）。
- [x] `airway generate scaffold <res> <field:type ...>`：一次产出带字段
  model、按 DSN 方言生成自增主键的迁移、service、JSON CRUD API、
  templ 页 + CRUD 岛屿（DataTable + Modal 表单，走 apiFetch），并
  **自动注册** config/routes.go（失败回退打印手工片段）。
- [x] `cmd/clitemplate` 同步：项目模板包含 `app/assets/` 骨架、预置
  `js.pkg.json`、带 `assets.Scripts()` 的 `base.templ`、示例岛屿；
  修改框架侧骨架时保持模板同步（现有惯例）。
- [x] `airway new` 的 Next steps 已提示 `airway js:install`
  （Phase 1 落地）。
- [x] 端到端实机验收通过（浏览器 CRUD 全流程，SQLite 持久化）；CI 化
  脚本 `.github/workflows/frontend.yml` 常驻。
  → 浏览器完成一次创建/编辑/删除。
- [x] CI 无 Node 验证 job：`.github/workflows/frontend.yml`
  （golang:1.26-alpine：new → js:install → scaffold → build →
  migrate → serve → probe）。

**验收**：全新项目、无 Node 环境，scaffold 产出可交互 CRUD 页面。

### Phase 5 结论（2026-09-17 执行，验收通过）

- 端到端实机（全程无 Node）：`airway new demo` → `js:install`（9 包）→
  `generate scaffold post title:string` → `go generate` → `js:build` →
  `db:migrate` → `server`，浏览器在 /posts 完成 Create/Update/Delete
  （四条 toast 对应，数据经 API 落 SQLite：seed → created → edited →
  deleted，最终表内容与操作一致）。
- scaffold 迁移的 id 列按当前 DSN 方言生成（BIGSERIAL /
  AUTO_INCREMENT / AUTOINCREMENT）。
- 顺带修复既有 bug：`generate service` 模板仍用旧版 lib/sql API
  （sql.InsertInto 等），已改为 generics API
  （repo.CreateFrom/UpdateByID/DeleteByID/FindByID）。
- 已知限制：框架 v0.9.0 发布前，新项目需
  `go mod edit -replace github.com/daqing/airway=<本地框架路径>` 才能
  用到 jsbuild/jspkg 等新包（Phase 6 发布后消除）；scaffold 重跑在
  文件已存在时报错退出（writeTemplateFile 的既有防覆盖语义）。

## Phase 6 — 文档与发布

**目标**：对外可用的完整交付。

任务：

- [x] `docs/frontend.md` + `docs/zh-CN/frontend.md`：全景、岛屿协议、
  开发流程、命令、组件清单、scaffold → 自定义岛屿教程。
- [x] README 定稿：Frontend strategy status 改为 implemented end to
  end（链接 frontend guide）；Features 增加 "Frontend without Node"
  条目；顶部文档链接加中英前端指南；AGENTS.md 项目布局自 Phase 2 起
  已逐步同步（assets/jsbuild/jspkg、岛屿与 airway-ui 约定、js 命令）。
- [x] CI 完整化：`ci.yml`（go build/vet/test 常驻，覆盖 jsbuild/
  jspkg/assets 全部单测）+ `frontend.yml`（golang:alpine 无 Node 端到
  端）。
- [ ] 版本发布：留给维护者执行（tag/push 是用户决策）。Phase 0–5 已
  全部落地，建议 bump VERSION 至 0.9.0 后发布——新项目的 jsbuild/
  jspkg 依赖只有发布后才能脱离本地 replace 正常 `go mod tidy`。

---

## 待决策（实施中需要拍板，默认值已标注）

1. **vendor/ 提交策略** — 默认提交（clone 即离线可构建）；代价是仓库
   体积，若不可接受改为 `js:install` 恢复 + CI 缓存。
2. **airway-ui 对用户项目的分发** — 默认 `js:add` 支持 GitHub 仓库来源
   （拉 release tarball），airway-ui 以
   `github.com/daqing/airway` 的子路径分发；替代方案是发独立 npm 包。
3. **样式体系** — 默认 CSS custom properties tokens + 组件级 CSS；
   不引入 Tailwind（需要构建期扫描）。若后期要 utility 类，做一个小型
   Go 侧原子类生成器，仍不引入 Node。
4. **showcase 路由（/ui）的开放范围** — 默认仅框架仓库 + local 环境可
   访问，不进用户项目二进制。
5. **`airway dev` 一体化命令**（server + esbuild watch + templ watch 一条
   命令拉起）— 默认 v1 不做，靠 `just dev`（Procfile）编排；呼声高再上。

### Phase 6 结论（2026-09-17 执行，文档与 CI 完成）

- 中英文前端指南落地；README 定稿（status/Features/文档链接）。
- CI 三件套：`ci.yml`（主测试）、`frontend.yml`（无 Node e2e）、
  既有 `docs.yml`。
- `go build` / `go vet` / `go test ./...` 全绿（28 包）。
- 前端计划全部完成，唯一未勾选项为版本发布（需维护者执行）。

## 里程碑建议

- **M1 = Phase 0–2**：前端工程地基（构建/依赖/dev 体验），发 v0.9.0。
- **M2 = Phase 3–4**：岛屿协议 + 组件库，框架能力完整。
- **M3 = Phase 5–6**：脚手架与文档，达到「airway new 即全栈可交互」。
