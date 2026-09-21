# 用 Wails 打包桌面应用

Airway 项目可以导出为 [Wails v3](https://v3.wails.io) 桌面应用：同一套 Web
技术栈（templ 视图、islands、JSON API、WebSocket）运行在桌面进程内的随机
`127.0.0.1` 端口上，原生 WebView 窗口直接加载该地址。服务端渲染、cookie
会话、重定向、WebSocket 的行为与线上完全一致——桌面壳是纯增量，应用代码
零改造。

调研与决策记录见仓库根目录的 [WAILS.md](../../WAILS.md)。

## 工作原理

```
┌─ 桌面二进制 ────────────────────────────────────────────┐
│ boot.New：用户数据目录中的 SQLite → 内嵌迁移 →           │
│           本地文件存储 → 插件 → 路由                     │
│ http.Server 监听 127.0.0.1:<随机端口>                    │
│ Wails v3 窗口 → 加载 http://127.0.0.1:<端口>             │
└─────────────────────────────────────────────────────────┘
```

- **数据库：** 纯 Go SQLite，位于 `<UserConfigDir>/<AppName>/data.db`
  （macOS 为 `~/Library/Application Support/<AppName>/`，Windows 为
  `%APPDATA%\<AppName>`）。`db/migrate` 中的迁移会内嵌进二进制，启动时
  自动执行。
- **文件存储：** local 驱动，根目录为
  `<UserConfigDir>/<AppName>/storage`。
- **安全加固：** 桌面启动固定 `AIRWAY_ENV=production`，直接使用已提交的
  前端 bundle（无需 Node、无开发服务器），把宽松的 CORS 中间件替换为仅同源
  策略，并把 WebSocket 升级限制为同源请求。
- **窗口就是真实的浏览器视图**，指向你自己的服务，因此同源请求
  （`apiFetch`）、cookie、302 重定向和 `ws://127.0.0.1:<端口>/ws`
  全部原样可用。

## 生成桌面目标

在项目根目录执行：

```bash
airway desktop:init
```

生成内容：

```
desktop/
  main.go            # 加载本地服务的 Wails 窗口（首次生成后归用户所有）
  migrations.go      # migrations 的 go:embed + 插件镜像（重跑时刷新）
  migrations/        # db/migrate/*.sql 的同步副本
  Taskfile.yml       # build / package / run / dev 入口
  build/             # 图标、plist、安装器配置（Wails 构建系统）
```

`desktop:init` 还会执行
`go get github.com/wailsapp/wails/v3@v3.0.0-beta.24`，把 Wails 模块固定到
项目 go.mod。重复执行是安全的：它只刷新 `desktop/migrations/` 与插件镜像，
不会改动 `desktop/main.go`（需要重新生成时加 `--force`）。

项目根目录 `plugins.go` 中声明的插件会以 blank import 的形式镜像到
`desktop/migrations.go`，桌面二进制与 Web 二进制编译同一批插件。用
`plugin:install` 添加插件后，重跑 `airway desktop:init` 即可。

## 构建与打包

先安装工具链（一次性）：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
go install github.com/go-task/task/v3/cmd/task@latest
```

然后在 `desktop/` 目录内：

```bash
wails3 task build        # 当前平台构建
wails3 task package      # macOS .app / Windows NSIS 安装器 / Linux deb+rpm
wails3 task dev          # 变更自动重编译的运行模式
```

产物位于 `desktop/bin/`。macOS 构建要求 macOS 12+；`.app` 默认 ad-hoc
签名——对外分发需要 Developer ID 签名与公证（`wails3 task darwin:sign` /
`darwin:sign:notarize`）。

### 平台说明

| 平台 | 要求 | 备注 |
|---|---|---|
| macOS 12+ | Xcode 命令行工具 | universal 二进制：`wails3 task darwin:package:universal` |
| Windows 10/11 | WebView2 运行时（NSIS 安装器自动安装） | 可从 macOS/Linux 免 CGO 交叉编译 |
| Linux | GTK4 + WebKitGTK 6.0（Ubuntu 24.04+/Debian 13+）；旧栈用 `-tags gtk3` | deb/rpm/AppImage 打包基于 nfpm |

macOS 与 Linux 的 CGO 交叉编译使用 Wails 官方 Docker 镜像
（`wails3 task setup:docker`，约 800MB）。正式发布推荐 GitHub Actions
三平台矩阵（每个平台一个 job）。

## 数据与迁移语义

- `desktop/migrations/` 是 `db/migrate/` 的**副本**。重跑
  `airway desktop:init` 会重新同步；从源目录删除的迁移会**保留**在副本中
  （该命令从不删除），保证老桌面安装仍能找到它们。
- 每次启动自动执行迁移；`schema_migrations` 记录已应用版本，新机器首次
  启动即可建出完整 schema。
- 桌面运行不会写 `db/schema.json`（`boot.Options` 中关闭了快照 IO）。

## 限制

- 在 Wails v3 到达 GA 之前属于实验性能力；Wails 版本已固定，升级需谨慎。
- 本地端口对同用户的其他进程可达——与所有"本地服务型"桌面应用相同的信任
  模型。已启用同源加固，但不要假设端口是秘密。
- 页面里的外链 `https://` 会接管桌面窗口本身；如不希望，请在 UI 代码里把
  外链转交系统浏览器打开。
