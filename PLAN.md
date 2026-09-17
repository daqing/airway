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

- [ ] 定义 `js.pkg.json`：依赖清单，**精确版本**（lock 语义，无 semver
  range 解析，v1 不做 ranges）。
- [ ] `cmd/jspkg` 实现 `airway js:add <pkg>[@<version>]`：从 registry 取
  元数据、下载 `dist.tarball`、解包（gzipped tar，纯 Go）到
  `app/assets/js/vendor/`（node_modules 兼容布局，保留包内 package.json
  供 esbuild 解析 exports/main），并写入 `js.pkg.json`。
- [ ] `airway js:install`：按 `js.pkg.json` 幂等重装（本地已有且校验
  一致则跳过）。
- [ ] registry 可配置：`AIRWAY_JS_REGISTRY`（默认
  `https://registry.npmjs.org`，可切 npmmirror 等镜像）。
- [ ] 基础依赖预置：`airway new` 生成的项目 `js.pkg.json` 预置 preact、
  preact/compat、`@tanstack/preact-query`、`react-hook-form`、
  airway-ui 基础包（对齐「前端库支持矩阵」）。
- [ ] 单测：tarball 解包、精确版本匹配、镜像 URL 拼接、幂等安装
  （本地 fixture，不依赖网络）。
- [ ] 决策并落地：`vendor/` 提交进仓库（默认：提交，保证 clone 即可
  离线构建；缺点是仓库变大）。

**验收**：`airway js:add @tanstack/react-table@<版本>` 后，TS 里
`import ... from "@tanstack/react-table"` 能被 js:build 成功解析。

## Phase 2 — 构建管线与开发流程

**目标**：`airway js:build` 出产物、dev 时内存构建、livereload、
`go:embed` 单二进制，前端开发闭环成立。

任务：

- [ ] `cmd/jsbuild` 实现 `airway js:build`：
  入口 `app/assets/js/app.tsx`，`vendor/` 为 node 解析路径，
  统一注入 `react`→`preact/compat` 别名；minify、sourcemap、
  `Target: ES2020`。
- [ ] 固定文件名（`app.js`）+ `manifest.json`（入口、构建 hash、chunk
  列表）；缓存策略用查询串 `?v=<hash>`，避免 hash 文件名导致每次构建
  大 diff。
- [ ] 产物写入 `app/assets/dist/` 并提交；`internal/assets` 用
  `//go:embed all:dist` 暴露 `http.Handler`，gin 挂 `/assets/*`（含
  `URL_PREFIX` 前缀适配）。
- [ ] dev 内存构建：`AIRWAY_ENV=local` 时 `/assets/*` 由 middleware 调
  esbuild `Rebuild`（按需 + 短 TTL）直接返回字节，ETag 为构建 hash；
  esbuild Watch 放 goroutine。
- [ ] livereload：esbuild v0.28 的 `Watch` 不对外暴露重建事件（Phase 0
  发现），需自研轻量源文件监听（mtime 轮询即可）触发 WebSocket hub
  （`app/websocket`，复用）广播；`base.templ` 在 local 下注入
  livereload 客户端脚本（整页刷新，v1 不做模块级 HMR）。
- [ ] `Procfile.dev` / `justfile` 联动：`just dev` 下 JS 重建 + Go 重启
  共存；`just build` 前置 `airway js:build`。
- [ ] 单测：manifest 生成、embed handler（httptest）、dev middleware
  缓存头、`URL_PREFIX` 前缀。

**验收**：改一个 `.tsx` 保存，浏览器约 1 秒内自动刷新；`go build` 出的
二进制拷到无源码机器可正常 serve `/assets/app.js`。

## Phase 3 — 岛屿运行时（templ ↔ Preact 协议）

**目标**：定义并实现挂载协议，让 templ 页面以声明式方式嵌入岛屿。

任务：

- [ ] 协议定稿：`<div data-island="OrderList" data-island-id="<n>">`
  挂载点 + 相邻 `<script type="application/json" id="island-data-<n>">`
  初始数据（由 `templ.JSONScript` 生成，处理 `</script>` 转义）。
- [ ] runtime：`app.tsx` 扫描 `[data-island]`，按注册表取出组件、解析
  props、`render()` 挂载；未注册/失败时 console 明确报错。
- [ ] 注册表代码生成：esbuild plugin（Go 侧 OnLoad）扫描
  `app/assets/js/islands/` 自动生成 `registry.gen.ts`，新增岛屿零手工
  注册。
- [ ] templ helper：`lib/island`（或 app 内共享包）
  `island.Component(name string, props any) templ.Component` —— 输出挂载
  点 + JSONScript；生产读 `dist/manifest.json` 注入 `<script src>`，
  local 注入 dev 入口 + livereload。
- [ ] `base.templ` 集成：head 尾部 `assets.Scripts()` helper，页面无感。
- [ ] demo：home 页加一个 counter 岛屿（props 由 action 传入），作为
  协议参考实现。
- [ ] 单测：helper 渲染断言（挂载点、JSON 转义、script 注入的
  生产/local 两种形态）。

**验收**：在一个现有 templ 页面里写 `{{ island.Component("Counter",
map[string]any{"start": 3}) }}`（等价 Go 调用），页面出现可交互组件；
禁用 JS 时页面骨架仍完整。

## Phase 4 — airway-ui 组件库

**目标**：CRUD 场景组件齐备的组件库底座：视觉层自研、逻辑层借
preact/compat 生态。

任务：

- [ ] design tokens：`css/airway.css` 用 CSS custom properties 定义色板、
  间距、字号、圆角、阴影、暗色变量；组件零第三方 CSS 依赖。
- [ ] 组件 API 风格定稿：TS props 类型、受控/非受控约定、事件命名、
  className 合并规则。
- [ ] 基础组件：Button、Input、Textarea、Select、Checkbox、Radio、
  Field（表单布局）、Spinner、EmptyState。
- [ ] Form：React Hook Form 集成封装（校验、错误展示、提交状态）——
  表单选型定为 RHF（官方支持 Preact），不用 TanStack Form。
- [ ] Table：`@tanstack/react-table`（compat）集成（排序、分页、行
  选择、空态/加载态）；出现兼容缺口时改为对 `@tanstack/table-core`
  自写 adapter。
- [ ] 反馈组件：Modal、Toast（消息队列）、Tabs、Pagination。
- [ ] 数据层：基于 `@tanstack/preact-query`（官方 Preact 适配），
  `ui/data` 封装 fetch 对齐 `lib/render` 的 ok/error JSON 约定（统一
  错误提示、401/500 处理钩子）。
- [ ] showcase：`/ui` 路由的组件演示页（templ + 岛屿，dogfooding，
  同时作为组件的可视化测试）。
- [ ] 无障碍基线：label 关联、键盘导航（Modal 焦点圈、Tab 顺序）。

**验收**：仅用 airway-ui 组件即可拼出一个完整的列表 + 新建/编辑 +
删除的 CRUD 页面；showcase 页所有组件可交互。

## Phase 5 — 脚手架整合

**目标**：generate 系列与 `airway new` 模板产出前端代码，scaffold 的
默认产物即是可交互页面。

任务：

- [ ] `airway generate island <name>`：生成 `islands/<name>.tsx` 骨架
  （props 类型 + demo 渲染），自动进入注册表。
- [ ] `airway generate scaffold <res> <field:type ...>`：在现有产出
  （model + migration + service + JSON API）之上，新增 templ 列表页
  （Table 岛屿）与新建/编辑页（Form 岛屿），props 从 action 传入。
- [ ] `cmd/clitemplate` 同步：项目模板包含 `app/assets/` 骨架、预置
  `js.pkg.json`、带 `assets.Scripts()` 的 `base.templ`、示例岛屿；
  修改框架侧骨架时保持模板同步（现有惯例）。
- [ ] `airway new` 产出后开箱即跑：README 的 Quick start 增加一步
  `airway js:install`（或 new 时自动完成）。
- [ ] 端到端手验（作为验收脚本写进 docs）：
  `airway new demo && cd demo && airway js:install && airway generate
  scaffold posts title:string && airway db:migrate && airway server`
  → 浏览器完成一次创建/编辑/删除。
- [ ] CI 无 Node 验证 job：`golang:alpine` 容器内跑通上述流程。

**验收**：全新项目、无 Node 环境，scaffold 产出可交互 CRUD 页面。

## Phase 6 — 文档与发布

**目标**：对外可用的完整交付。

任务：

- [ ] `docs/frontend.md` + `docs/zh-CN/frontend.md`：概念（岛屿、协议）、
  命令、组件清单、从 scaffold 到自定义岛屿的教程。
- [ ] README 更新：移除 Frontend strategy 的 "not yet implemented" 标注，
  Features 与 Quick start 纳入前端流程；AGENTS.md 项目布局同步。
- [ ] CI 完整化：单测（jsbuild/jspkg/middleware/helper）+ 无 Node 构建
  job 常驻。
- [ ] 版本发布：建议 Phase 1–2 合入后发 v0.9.0，Phase 5 合入后发
  v1.0 候选。

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

## 里程碑建议

- **M1 = Phase 0–2**：前端工程地基（构建/依赖/dev 体验），发 v0.9.0。
- **M2 = Phase 3–4**：岛屿协议 + 组件库，框架能力完整。
- **M3 = Phase 5–6**：脚手架与文档，达到「airway new 即全栈可交互」。
