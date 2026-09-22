# Wails 桌面化可行性调研：把 Airway 应用导出为跨平台桌面软件

> 调研日期：2026-09-21。事实核对基于 wails.io / v3.wails.io 官方文档、
> github.com/wailsapp/wails 源码与 Releases（v2.14.0、v3.0.0-beta.24）以及本仓库代码。

## 结论（TL;DR）

**可行，且已通过本机 POC 实测（2026-09-21，验证项全绿，见 §9）。**
推荐：**Wails v3（当前 beta）+ 「窗口加载本机 HTTP 服务」模式**。
Airway 应用进程内启动 Gin 引擎并监听 `127.0.0.1:<随机端口>`，Wails 桌面窗口直接加载该地址。
templ 服务端渲染、Preact islands、cookie 会话、302 重定向、WebSocket 全部原样工作，
**应用代码零改造**。

代价与前置条件：

1. 框架侧需要一次小规模重构：把 `App`（`app.go`）和迁移管理器（`cmd/cli_migrate.go`）
   抽成可导入的包，并提供**显式参数**（而非环境变量）的编程式启动接口（原因见 §6.2）。
2. v3 尚在 beta（桌面 API 已由官方声明稳定，已有团队生产使用），生成器应**锁定精确版本**
   并标记 experimental，跟进 RC/GA。
3. 分发侧：Windows 可免 CGO 交叉编译；macOS/Linux 走 CI 或官方 Docker 交叉构建；
   macOS 公开分发必须签名 + 公证；v3 桌面模式需要本机回环端口（仅 127.0.0.1 可达，
   配合加固清单可接受）。

不推荐的路线：Wails v2 的 AssetServer 管道（cookie/302/WebSocket 三项硬伤，见 §4）、
把前端重写为 Wails bindings（背离"自动导出"目标）。

---

## 1. Wails 现状（2026-09）

| | Wails v2 | Wails v3 |
|---|---|---|
| 最新版本 | v2.14.0（2026-08-10） | v3.0.0-beta.24（2026-09-20） |
| 状态 | 稳定版，**仅修复模式**（每周自动切补丁版本） | Beta（2026-08-02 进入），桌面 API 官方声明稳定，有团队生产使用，RC→GA 路线已公布 |
| 模块路径 | `github.com/wailsapp/wails/v2` | `github.com/wailsapp/wails/v3` |
| 架构 | OS 原生 WebView（Windows: WebView2 / macOS: WKWebView / Linux: WebKitGTK）+ Go 后端，单二进制 | 同左，重写：多窗口、Service 模型、静态分析绑定、Taskfile 构建系统 |
| 窗口加载外部 URL | **不支持**（生产模式起始 URL 硬编码为 `wails://wails/` 或 `http://wails.localhost/`） | 支持：`WebviewWindowOptions.URL` / `Window.SetURL` |
| 许可证 | MIT | MIT |

项目健康度：约 36.3k stars，主维护者 leaanthony（4k+ commits），v3 beta 7 周内发了 24 个
tag，节奏很快；官方博客 candid 地承认 v3 alpha 阶段偏长，beta 起引入 WEP 提案流程与明确
的稳定性边界。参考资料见 §11。

对 Airway 的含义：**v2 是维护线，v3 是投入线**。而 Airway 的应用形态（服务端渲染 +
cookie 会话 + WebSocket）恰好是 v2 架构的弱项、v3 的强项（§4），因此调研结论指向 v3。

## 2. Airway 架构中与桌面化相关的事实

以下均为本仓库代码核实的事实：

| # | 事实 | 位置 | 对桌面化的影响 |
|---|---|---|---|
| 1 | `App.Handler()` 导出标准 `http.Handler`（含 URL_PREFIX 剥离逻辑） | `app.go:54` | 整个应用可作为普通 http.Handler 被桌面壳托管 |
| 2 | 启动序列线性且各步可复用：.env → SetupDB → Redis(可选) → storage → jsbuild(仅 local) → plugin.BootAll → Listen | `main.go:57` `runServer` | 桌面进程可复刻同一序列，跳过 Redis 与 jsbuild |
| 3 | DSN 读取顺序 `AIRWAY_DSN → DSN → AIRWAY_DB_DSN → AIRWAY_PG`，驱动由 DSN scheme 推断 | `cmd/cli.go:78` | 桌面端指向 `sqlite://<用户数据目录>/app.db` 即可单机运行 |
| 4 | SQLite 驱动为 modernc 纯 Go 实现 | `go.mod` | 桌面单机免外部数据库、免 CGO 数据库依赖 |
| 5 | 迁移管理器是 `cmd` 包内未导出实现，路径写死 `./db/migrate` 与 `./db/schema.json` | `cmd/cli_migrate.go` | 需重构导出为编程式 API，且支持内嵌迁移目录（`go:embed` 不能引用 `..`） |
| 6 | 根目录为 `package main`（`main.go` + `app.go`），不可被导入 | 仓库根 | 桌面包装器必须放在子包（如 `desktop/`），因此 App 构建逻辑需先抽到可导入包 |
| 7 | 前端 `apiFetch` 全部使用相对路径（同源请求）；前端源码尚无 WebSocket 客户端调用（`/ws` 目前仅 Go 侧 hub） | `app/assets/js/ui/data.ts:27` | 窗口源 = 服务源时前端完全透明 |
| 8 | 生产模式前端 bundle 已提交并通过 `go:embed` 内嵌（`js:build` 无需 Node） | `app/assets/assets.go` | 桌面二进制天然携带前端，wails 构建的 frontend 命令可置空或调 `go run . js:build` |
| 9 | `AIRWAY_ENV != local` 时 Gin 进 release 模式、跳过 livereload | `lib/utils/config.go` | 桌面端固定设非 local 即可 |
| 10 | CORS 中间件默认 AllowAllOrigins + credentials；ws upgrader CheckOrigin 全放行 | `app.go:104`、`app/websocket/conn.go` | 桌面构建需收紧（§8） |
| 11 | 迁移不会随 server 启动自动执行（独立 CLI 命令） | `cmd/cli_migrate.go` | 桌面端首次启动需进程内自动迁移 |
| 12 | `airway new` 项目模板与框架骨架存在镜像同步约定 | `cmd/clitemplate/`、AGENTS.md | `desktop:init` 生成器需同步维护 clitemplate 模板 |

## 3. 集成模式对比

四条候选路线。"改造量"指对**用户应用代码**的改动。

### 模式 A：v3 `Assets.Handler`（Gin 引擎直挂，不开端口）

v3 官方一等支持：`application.Options.Assets.Handler = ginEngine`（配合 `GinMiddleware`
放行非 `/wails` 前缀请求），官方仓库自带 gin-example/gin-service 示例。优点是
"不开端口"；缺点是请求走注入的自定义协议管道（`wails://` / `http://wails.localhost`），
该管道**不承载 WebSocket**（官方把 WS 需求导向 Streams API），cookie 在自定义 scheme 上
沿用上游限制（v2 上已确认生产模式 macOS 失效），Linux 需 `webkit2_36/40` build tag 才
支持非 GET body。**对 Airway 不适配**（cookie 会话 + `/ws`），仅适合纯 API/静态前端。

### 模式 B：v3 窗口加载 `http://127.0.0.1:<port>`（推荐）

`WebviewWindowOptions.URL` 直接指向进程内启动的 Airway 服务。官方 Streams 指南明确承认
这一形态（"app stands up its own http.Server on a local port and the frontend connects
back to it"）。

- 源 = `http://127.0.0.1:<port>`：**同源** fetch/cookie/302/`ws://` 全部是标准浏览器行为，
  前端与服务端零改动；
- 端口暴露面仅本机回环 + 随机端口；加固清单见 §8；
- 官方文档同时给出了该模式的成本提示：需处理 CORS（我们同源化后不存在）与 localhost
  令牌（可选加固项）。

### 模式 C：v2 `AssetServer.Handler`（nil Assets + Gin 引擎兜底）

v2 的文档化模式：`assetserver.Options{Handler: yourGinEngine}`（Assets 为 nil 时全部
GET 进 Handler，非 GET 恒进 Handler）。但 v2 **没有窗口加载外部 URL 的选项**，一切请求
都走注入管道，官方特性矩阵给出硬约束：

| 特性 | AssetServer 管道表现（v2） | 对 Airway 的影响 |
|---|---|---|
| WebSocket | ❌ 全平台 | `/ws` 必须拆到第二个真实监听器，前端 WS URL 也要按环境切换 |
| Cookie | 自定义 scheme 上生产模式 macOS 失效（官方 issue #2590/#3908，无上游解法） | admin 面板的 cookie 会话在 mac 打包版不可用 |
| 30x 重定向 | Windows ✅ / macOS ❌ / Linux ❌ | 登录跳转等 templ 流程在 mac/Linux 需重写 |
| 响应体流式 | Windows ❌ | 当前无 SSE 场景，影响有限 |

结论：v2 路线要求应用层改造认证与 WS，违背"自动导出"目标，**不推荐**。

### 模式 D：Wails bindings 重写

把 API 层改写为 `Bind`/`Service` 方法调用 + 自动生成 TS 绑定。这是 Wails 的"正统"用法，
但需要重写每个 API 模块与前端数据层，templ SSR 一并废弃——不是"导出"，是重写。排除。

### 对比总表

| | A：v3 Handler 直挂 | **B：v3 窗口加载 localhost（推荐）** | C：v2 Handler 兜底 | D：bindings 重写 |
|---|---|---|---|---|
| 应用代码改造 | 中（WS 改 Streams） | **无** | 大（认证+WS） | 重写级 |
| templ SSR 保留 | ✅ | ✅ | ✅ | ❌ |
| cookie 会话 | 受自定义 scheme 限制 | ✅ 原生 | ❌（mac 生产） | N/A |
| 302 重定向 | 受管道限制 | ✅ 原生 | ❌（mac/Linux） | N/A |
| WebSocket | ❌（走 Streams） | ✅ 原生 `ws://127.0.0.1` | ❌ 需第二监听器 | 走 v3 events |
| 回环端口 | 无 | 有（仅 127.0.0.1） | 无（WS 除外） | 无 |

## 4. 推荐方案详解（模式 B）

桌面应用 = 同一个 Go 二进制里跑两件事：

```
┌─ desktop 二进制 ──────────────────────────────────────┐
│  boot: env 注入 → sqlite(userData) → 迁移 → storage    │
│        → plugins → gin.Handler                        │
│  net.Listen("tcp","127.0.0.1:0") → http.Server.Serve  │
│  Wails v3: application.New → Window(URL=该地址) → Run │
└───────────────────────────────────────────────────────┘
```

- **数据落盘**：`os.UserConfigDir()` 下（macOS `~/Library/Application Support/<App>/`，
  Windows `%APPDATA%\<App>\`，Linux `~/.local/share/<App>/`）：`data.db`（SQLite）、
  `storage/`（本地文件存储根）。
- **端口**：`127.0.0.1:0` 随机分配，规避端口冲突；地址只取 `ln.Addr()`。
- **迁移**：进程内自动执行（幂等，`schema_migrations` 表），迁移 SQL 内嵌进二进制。
- **Redis / jsbuild dev server / URL_PREFIX**：桌面构建一律跳过/置空。
- **schema:dump**：桌面运行时关闭（避免往安装目录写 `db/schema.json`）。

### 与 Airway 既有机制的契合点

- 前端 bundle 已内嵌（`app/assets/dist` 已提交 + `go:embed`），桌面二进制天然离线可用；
  wails 构建的 frontend 步骤直接复用，**保持 Node-free**（jspkg/jsbuild 优势）。
- `*_templ.go` 已提交，桌面构建不需要 templ CLI。
- v3 还有 `-tags server` 构建模式：同一份代码可编译为纯 HTTP 服务（无窗口），与 Airway
  的 Web 部署形态天然互补（同一应用既是网站又是桌面程序）。

## 5. 「自动导出」落地设计：`airway desktop:init`

目标形态：**同一仓库内新增 `desktop/` 子包**（不是独立导出项目）。理由：
用户模型/服务/API/视图都在本项目内，独立项目反而要倒导模块路径；同仓库共享 go.mod 与
版本号，`go build ./desktop` 即参与 `go vet ./...`/`go test ./...`。

```
desktop/
  main.go            # package main，见 §5.2
  migrations/        # desktop:build 前从 db/migrate 同步（脚本/命令负责），go:embed all:migrations
  migrations.go      # //go:embed 声明
  Taskfile.yml       # wails3 构建任务（wails3 init 产物定制）
  build/             # 图标、Info.plist、Windows 安装器配置、Linux 打包配置
```

### 5.1 框架前置重构（✅ 已实现：R1 `lib/app`、R2 `lib/migrate`、R3 `lib/boot.New`、R4 StrictCORS/SameOriginWS）

| # | 重构 | 状态 |
|---|---|---|
| R1 | `App`/`CORS`/`prefixHandler` 抽到 `lib/app`（`NewApp(name, port, opts...)` + `WithCORS` + `StrictOrigin`） | ✅ |
| R2 | 迁移引擎导出为 `lib/migrate`（`Run/RunTo/Rollback/Status` over `Options{DSN, Migrations fs.FS, SnapshotPath, Out}`），CLI 变薄壳 | ✅ |
| R3 | `lib/boot.New(Options)` 统一启动入口，**显式传参**（Env/DSN/Migrations/StorageRoot/RedisURL），不复用环境变量注入 | ✅ |
| R4 | 桌面开关：`StrictCORS`（仅同源）、`SameOriginWS`（`websocket.CheckSameOrigin()`）、关闭 schema snapshot | ✅ |

### 5.2 关键陷阱：环境变量快照优先级

`lib/utils` 的多键读取会**优先查进程启动时的环境快照**（`shellEnv`，`lib/utils/env.go`），
其次才查当前 `os.Getenv`。如果桌面包装器靠 `os.Setenv("AIRWAY_DSN", ...)` 注入配置，
用户从终端带 `AIRWAY_DSN` 启动应用时会**静默用到 shell 里的值**。因此 R3 的桌面启动接口
必须**显式传参**（DSN、StorageRoot 等直接作为函数参数进入 `repo.SetupDB`/`storage.Setup`），
或者在注入前对相关键做 `os.Unsetenv`。这是"导出器"最容易踩的隐性坑。

### 5.3 生成的 `desktop/main.go`（已实现，节选）

实际生成器为 `airway desktop:init`（模板在 `cmd/clitemplate/desktop/`，
文档见 `docs/desktop.md` / `docs/zh-CN/desktop.md`）：

```go
package main

import (
    "net"
    "net/http"
    "os"
    "path/filepath"

    "github.com/wailsapp/wails/v3/pkg/application"

    "github.com/daqing/airway/lib/boot"          // R3
    "<project-module>/desktop/migrations"        // 内嵌迁移
)

func main() {
    root, _ := os.UserConfigDir()
    dataDir := filepath.Join(root, "MyApp")

    handler, err := boot.New(boot.Options{ // R3：显式传参，不注入环境变量
        AppName:    "MyApp",
        Env:        "production",
        DSN:        "sqlite://" + filepath.ToSlash(filepath.Join(dataDir, "data.db")),
        Migrations: migrations.FS,
        StorageRoot: filepath.Join(dataDir, "storage"),
        StrictCORS: true, // R4
        SameOriginWS: true,
    })
    if err != nil {
        application.Fatal(err) // 弹窗 + 退出
    }

    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        application.Fatal(err)
    }
    go (&http.Server{Handler: handler}).Serve(ln)

    app := application.New(application.Options{Name: "MyApp"})
    app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "MyApp",
        Width:  1280, Height: 800,
        URL:    "http://" + ln.Addr().String(),
    })
    app.Run()
}
```

`desktop:init` 的生成逻辑与 `admin:generate` 同风格：从 `go.mod` 读模块路径与应用名，
按 clitemplate 模板渲染 `desktop/`（模板同样镜像进 `cmd/clitemplate/`），并提示安装
`wails3` CLI 与 `task`。生成器本身不执行构建。

### 5.4 运行时行为清单

| 关注点 | 行为 |
|---|---|
| 单实例 | v2 有 `SingleInstanceLock`；v3 对应能力**待 POC 确认**（§9） |
| 外链 | 页面内 `https://` 外链会接管窗口，需约定用 `runtime.BrowserOpenURL`（或在服务端拦 no-op） |
| 窗口关闭 | `http.Server` 随进程退出；如需托盘常驻可后置（v3 SystemTray 一等支持） |
| 自动更新 | v2 无内置；v3 声明带 updater（细节待 POC 验证），过渡期可"替换二进制 + 重启" |

## 6. 构建、分发与 CI

### 6.1 平台矩阵

| 目标 | 构建主机 | 说明 |
|---|---|---|
| macOS (Intel+Apple Silicon, universal) | macOS（CI: `macos-latest`） | `wails3 task darwin:build:universal`；beta.24 的部署目标为 **macOS 12+**（Taskfile `MACOSX_DEPLOYMENT_TARGET=12.0`）；交叉编译出的 mac 包**不带签名** |
| Windows amd64/arm64 | 任意主机 | `GOOS=windows` 交叉编译**无需 CGO**（v3 文档明确）；NSIS 安装器含 WebView2 bootstrapper，安装器生成建议在 Windows CI 上做 |
| Linux amd64/arm64 | Linux（CI: `ubuntu-latest`） | 默认 GTK4 + WebKitGTK 6.0；老发行版用 `-tags gtk3`（v3.0.x 支持，v3.1 移除）；v3 提供打包指南（deb/rpm） |
| 单机交叉（可选） | Docker | `wails3 task setup:docker`（~800MB 镜像）覆盖 CGO 构建的交叉场景 |

生产推荐 **GitHub Actions 三平台矩阵**（v2/v3 官方文档均为此路线），macOS runner 上完成
签名 + 公证（Developer ID Application 证书 + notarytool；entitlements 需含 network
client/server）。

### 6.2 体积与依赖预期

Wails 官方口径：二进制约 15MB、基线内存约 10MB（对比 Electron ~150MB）。Airway 叠加
gin + pgx + mysql + modernc/sqlite + goldmark + minio 后预计 **25–35MB**，内存表现仍远优
于 Electron。Windows 首次运行依赖 WebView2 runtime（Win10/11 大多预装；安装器可内嵌
bootstrapper 或固定版本运行时）。

## 7. 安全与加固清单（模式 B 专属）

1. **只绑 127.0.0.1 + 随机端口**（方案固有）。同机其他进程可达——这是所有"本地服务型"
   桌面应用（Electron/Steam/Docker Desktop 等）的共性，接受；敏感操作可加启动期随机
   令牌头（需要前端配合，属可选增强）。
2. **CORS 收紧**：桌面构建不再需要 `AllowAllOrigins`（同源化后无跨域）；boot 桌面开关
   直接替换为拒绝跨域的中间件（R4）。
3. **ws CheckOrigin 收紧**：同源化后可校验 Origin 为 `http://127.0.0.1:<port>`。
4. **Cookie**：显式设置 `SameSite=Lax`（现代 WebView 默认即 Lax，但应显式声明），
   配合 Secure 属性不适用于 http 环境可不设；admin 会话机制不变。
5. **外发链接**走系统浏览器，避免桌面窗口变成通用浏览器。
6. **迁移与数据文件**均在用户数据目录，注意权限 0700/0600。

## 8. 风险与待验证项

| 风险 | 等级 | 缓解 |
|---|---|---|
| v3 仍是 beta，API 可能变动 | 中 | 生成器锁定精确版本（go.mod pin）；标记 experimental；跟进 RC/GA 再转正 |
| GA 时间未定 | 低 | 官方已公布 RC→GA 里程碑；beta.24 起每 1–4 天一个 tag，节奏健康 |
| v3 单实例/自动更新等具体 API 未逐条核实 | 中 | 列入后续验证清单（§5.4）；签名/公证/DMG 已确认有内置任务（§9） |
| 模板构建链把前端绑死 npm/Vite | 低 | POC 已复现（模板 vite 构建失败，与 Wails 本体无关）；`desktop:init` 重写 Taskfile 前端任务为 `go run . js:build`（§9） |
| Linux WebKitGTK 6.0 在老发行版缺失 | 低 | `-tags gtk3` 兜底（至 v3.1）+ 文档写明最低发行版 |
| macOS 公证流程 | 低 | 标准流程，CI 固化；不经 App Store 分发无审核 |
| 仓库 `db/migrate` → `desktop/migrations` 同步遗漏 | 低 | 同步逻辑挂在 desktop:build/justfile 钩子；或 R2 直接支持 `overlay fs` 合并 |

## 9. 落地路线图（含 POC）

**第 0 步：独立 POC —— ✅ 已完成（2026-09-21，darwin/arm64，wails3 v3.0.0-beta.24）**

在 `/tmp/airway-poc` 用 `wails3 init`（vanilla 模板）生成项目后，把窗口改为加载
`http://127.0.0.1:<随机端口>` 的内置验证服务器（模拟 Airway：`/` 302 → `/home` +
Set-Cookie → 页面 JS 同源 fetch + WebSocket 往返 → 结果回传落盘）。结果：

| 验证项 | 结果 |
|---|---|
| 工具链（`go build -tags desktop,production`，CGO/clang） | ✅ 16MB 原生二进制 |
| 窗口加载 `http://127.0.0.1:<port>` | ✅ |
| 302 重定向被 WebView 正常跟随 | ✅（报告由 `/home` 页面 JS 生成，证明跳转链路通） |
| Cookie 设置 + `document.cookie` 可读 + 同源 fetch 自动携带 | ✅ `poc_token=abc123` 全链路一致 |
| WebSocket `ws://127.0.0.1:<port>/ws` 完整往返 | ✅ `ws-ok:hello-from-webview` |
| Host/Origin：同源导航，无 CORS 参与，Host 头完好 | ✅ |
| `GOOS=windows` 交叉编译（免 CGO） | ✅ 17MB exe |
| `.app` 打包 + ad-hoc 签名 + Info.plist | ✅ `wails3 task darwin:create:app:bundle` |

POC 过程中的额外发现（已修正正文相应章节）：

- beta.24 的 darwin 构建把 `MACOSX_DEPLOYMENT_TARGET` 定为 **12.0**（官方旧文档写
  10.15+，以 Taskfile 为准）。
- v3 内置签名/公证/DMG 工具：`darwin:sign`、`darwin:sign:notarize`（`wails3 tool sign`）、
  `create:dmg`；universal 二进制用 lipo 自动合并。比 v2 的 DIY 流程省事得多。
- 官方构建系统自带 `build:server` / `build:docker` 任务，`-tags server` 服务器模式是一等
  公民——与 Airway 的 Web 部署形态互补得到官方工具链支撑。
- 模板的完整 `task build` 链路把前端构建绑死在 npm/Vite 上（本机 vite/rolldown 构建失败，
  与 Wails 本体无关）。**`desktop:init` 必须重写 Taskfile 的前端任务**：改为
  `go run . js:build`（Airway 的 Node-free 管道）或直接用已提交的 dist。

**第 1 步：框架重构**（✅ R1–R4 已完成，`go test ./...` 全绿，`airway server` 行为不变）。

**第 2 步：`airway desktop:init` 生成器**（✅ 已完成：`cmd/cli_desktop.go` +
`cmd/clitemplate/desktop/` 模板镜像 + `docs/desktop.md`（英文 + zh-CN）；
`desktop/migrations.go` 同时镜像宿主 `plugins.go` 的 blank import，重跑保护
`main.go`、只刷新迁移与插件镜像）。

**第 3 步：CI 矩阵 + 签名/公证 + 安装器**（justfile 已加 `just desktop` /
`just desktop-package`；CI workflow 待建）。

**第 4 步：跟进 v3 RC/GA**，转正 experimental 标记。

## 10. 备选壳方案简评（为什么不是它们）

| 方案 | 现状 | 评语 |
|---|---|---|
| Tauri v2（v2.11.6） | Rust 核 + 系统 WebView | WebView 可加载外部 URL（`WebviewUrl::External`），但模型假定静态 `frontendDist`；Go 服务只能作为 sidecar 子进程管理，完全服务端渲染的应用是二等公民，还需 remote-capability 配置才能用 IPC。可行但别扭 |
| Electron（v44.x） | 活跃 | `child_process` 拉起 Go 二进制 + `loadURL('http://127.0.0.1:...')` 即通，保真度最高；代价是体积/内存（~150MB 级）与 Node 工具链，且 Go 侧得不到 Wails 的 Go 原生菜单/托盘/对话框 |
| Neutralino（v6.9） | 活跃 | `url` 配置指向任意 URL 即可，但无 Go 集成，进程管理/打包/更新全 DIY |
| 裸 WebView 绑定（go-webview2 / webview_go） | 维护弱 | 一个窗口指到 localhost，最轻；但菜单、托盘、安装器、签名、更新全要自造 |

结论：在"保留完整 Go Web 应用、只加壳"的命题下，Wails（尤其 v3）是唯一同时给出
**Go 原生集成 + 打包/签名工具链 + 活跃维护**的选项；Electron 是唯一在保真度上打平的
替代，代价是资源占用与跨语言工具链。

## 11. 参考资料

- Wails v3 Beta 公告（状态/治理/路线）：https://v3.wails.io/blog/wails-v3-beta
- Wails Releases（v2.14.0 / v3.0.0-beta.24）：https://github.com/wailsapp/wails/releases
- v3 Gin 集成指南：https://v3.wails.io/guides/gin-routing 、gin-service：https://v3.wails.io/guides/gin-services
- v3 Streams（WS 迁移与 localhost 服务器形态）：https://v3.wails.io/guides/streams-from-websockets
- v3 窗口选项（URL 字段）：https://github.com/wailsapp/wails/blob/master/v3/pkg/application/webview_window_options.go
- v3 服务器构建模式（`-tags server`）：https://v3.wails.io/guides/server-build
- v3 交叉编译：https://v3.wails.io/guides/build/cross-platform ；Linux 构建：https://v3.wails.io/guides/build/linux
- v2 AssetServer 语义与特性矩阵：https://wails.io/docs/reference/options#assetserver
- v2 cookie 限制（issue）：https://github.com/wailsapp/wails/issues/2590 、https://github.com/wailsapp/wails/issues/3908
- v2 签名指南：https://wails.io/docs/guides/signing ；跨平台构建：https://wails.io/docs/guides/crossplatform-build
- v2→v3 迁移指南：https://v3.wails.io/migration/v2-to-v3/
- 社区先例（Wails + templ + Gin）：https://github.com/dctellya/wails-htmx-templ-gin-template
- 官方 Gin 示例：https://github.com/wailsapp/wails/tree/master/v3/examples
