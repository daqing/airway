# Airway

Airway 是一个受 Rails 启发的 Go 全栈框架。它用同一套应用代码构建服务端渲染的
Web 应用、JSON API、静态展示站点和原生桌面应用，支持 **PostgreSQL**、
**MySQL 8** 和 **SQLite**——数据库驱动在运行时根据 DSN 自动推断。

- **[English README](../../README.md)**
- **[Admin 后台指南](../../ADMIN.zh-CN.md)** / **[Admin panel guide](../../ADMIN.md)**
- **[静态展示站点（SSG）](../../SSG.zh-CN.md)** / **[Static showcase sites](../../SSG.md)**
- **[静态导出指南](static-export.md)** / **[Static export guide](../static-export.md)**
- **[CLI 脚手架指南](cli.md)** / **[CLI scaffolding guide](../cli.md)**
- **[前端指南](frontend.md)** / **[Frontend guide](../frontend.md)**
- **[OpenAPI 指南](openapi.md)** / **[OpenAPI guide](../openapi.md)**
- **[桌面应用指南](desktop.md)** / **[Desktop guide](../desktop.md)**
- **[Plugin 扩展机制](plugin.md)** / **[Plugin guide](../plugin.md)**
- **[文件存储指南](storage.md)** / **[Storage guide](../storage.md)**
- **[视图模板指南（templ）](template.md)** / **[templ views guide](../template.md)**
- **[SQL Builder DSL 指南](sql-builder.md)**

## 特性

- **基于泛型的 Repository**（`lib/repo`）：类型安全的 `FindBy[User]`、`CreateFrom[User]`、预加载（eager loading）、联表查询、事务绑定助手。
- **方言感知的 SQL Builder**（`lib/sql` + `pg` / `mysql` / `sqlite` 方言包）；条件写法如 `sql.Eq`、`sql.AllOf`、`sql.Gt`；原生支持 `FOR UPDATE SKIP LOCKED`（`sql.ForUpdateSkipLocked`）。
- **基于 schema 的迁移**：通过 CLI 生成与应用迁移；SQLite 的表结构变更通过重建表处理。
- **统一的文件存储**（`lib/storage`）：本地目录、Amazon S3、Cloudflare R2 或腾讯云 COS——完全由配置决定，并提供 HTTP 上传/下载 API。
- **Gin Web 服务 + WebSocket** 发布/订阅。
- **基于 [templ](https://templ.guide/) 的 HTML 视图**：页面是 `app/views/` 下的 `.templ` 模板，由 action 经 `lib/render.HTML` 渲染。
- **无需 Node 的前端**：npm 依赖由 CLI 管理（`js:add` / `js:install`，`js.pkg.json` 锁定），内嵌 esbuild 打包（`js:build`），开发期内存重建 + livereload，以及构建在内置 airway-ui 组件库之上的交互式 **Preact islands**——产物全部提交并内嵌进单个 Go 二进制。
- **OpenAPI 3.2 文档**：从运行中的路由自动生成——operationId 由 handler 名推导，可在各模块中用代码富化，并实时服务于 `/openapi.json`。
- **Admin 后台生成器**：一份 TOML 表规格即可生成完整的后台——认证、角色、审计日志、CSV 导出、服务端分页列表。
- **桌面应用**：`airway desktop:init` 把项目导出为 Wails v3 桌面目标；同一套 Web 技术栈跑在 macOS、Windows、Linux 的原生 WebView 窗口里。
- **静态展示站点**：`airway ssg:new` 脚手架一个页面即 Go 代码的静态站点，外观来自可替换的主题 module（`theme:new` / `theme:install`）；`airway ssg:build` 导出纯 HTML，可部署到任何静态托管。
- **静态导出上 CDN**：`airway static:build` 把应用页面（在 `export.go` 中注册）导出为纯 HTML 加前端产物——templ 渲染的内容 + 完整保留的 Preact islands，可直接部署到任何 CDN（见 [docs/static-export.md](static-export.md)）。
- **插件**：WordPress 风格的功能模块，以独立 Go module 分发——`go get` 安装，`plugins.go` 里一行 blank import 启用（见 [docs/plugin.md](plugin.md)）。
- **脚手架 CLI**（`airway generate ...`、`db:migrate` 等）。
- **Repo REPL**：支持类型化扫描与 Go 表达式求值。
- **可选子路径前缀**（`URL_PREFIX`）：便于在反向代理后部署，例如 `http://host:1900/airway/...`。

## 快速开始

需要 Go **1.27.1 或更高版本**。无需 Node.js，无需 CGO。

### 1. 安装 CLI 并生成新项目

```bash
go install github.com/daqing/airway@latest
airway new myapp        # 或：airway new github.com/me/myapp（目录取路径最后一段）
                        # 或：airway new /path/to/myapp（传路径，module 取路径最后一段；相对路径同理，如 sites/myapp）
cd myapp
```

`airway new` 会以框架的 `app/` 骨架为模板生成一个新项目，自动从 `.env.example`
生成 `.env`，执行 `go mod tidy`，安装前端依赖（`airway js:install`），并打印
后续步骤。如果你是想开发框架本身，可以改为克隆仓库：

```bash
git clone https://github.com/daqing/airway.git
cd airway
cp .env.example .env
```

打开 `.env`，至少设置数据库 DSN 与端口：

```env
DSN="sqlite://./tmp/airway.db"     # 或 postgres://...、mysql://...
PORT="1900"
```

环境变量既可以使用短名（`DSN`、`PORT`、`REDIS`、`URL_PREFIX`），也可以使用带 `AIRWAY_` 前缀的形式（`AIRWAY_DSN`、`AIRWAY_PORT` 等）。当两者都设置时，以 `AIRWAY_` 形式为准。参见[「配置」](#配置)。

### 2. 启动服务

```bash
go run . server   # 或：airway server（需要 AIRWAY_ENV，如 AIRWAY_ENV=local，以及配置好的 DSN）
```

或者启动带热重载的本地开发服务：

```bash
just dev
```

应用监听 `http://127.0.0.1:1900`（`GET /` 返回 templ 渲染的 HTML 页面，`GET /health` 返回 `UP`）。

## 配置

所有配置通过环境变量完成（见 `.env.example`）。`.env` 中的值是**回退值：进程环境始终优先**（例如 `PORT=1988 airway server` 会覆盖 `.env` 里的 `PORT`）。每个值都可以用短名或其 `AIRWAY_` 别名，两者都设置时以 `AIRWAY_` 形式为准。

| 变量 | 说明 |
| --- | --- |
| `DSN` / `AIRWAY_DSN` | 数据库 URL，驱动由 scheme 推断（见下）。 |
| `PORT` / `AIRWAY_PORT` | HTTP 监听端口（默认 `1900`）。 |
| `REDIS` / `AIRWAY_REDIS` | 可选的 Redis URL，用于缓存/队列。 |
| `URL_PREFIX` / `AIRWAY_URL_PREFIX` | 可选的公开子路径前缀，如 `/airway`；留空则在根路径服务。 |
| `AIRWAY_JS_REGISTRY` | `js:add` / `js:install` 使用的 npm registry（默认 `https://registry.npmjs.org`；需要时可设为 `https://registry.npmmirror.com` 等镜像）。 |
| `AIRWAY_ENV` | `local` 会加载 `.env` 并使用 Gin debug 模式；其他值为 release 模式。 |
| `STORAGE_DRIVER` | `local`（默认）、`s3`、`r2` 或 `cos`。 |
| `STORAGE_ROOT` | 本地存储根目录（默认 `./data/storage`）。 |
| `STORAGE_*` | 云存储配置：`STORAGE_REGION`、`STORAGE_ENDPOINT`、`STORAGE_BUCKET`、`STORAGE_ACCESS_KEY`、`STORAGE_SECRET_KEY`、`STORAGE_PUBLIC_URL`（可选 CDN 基址）。 |
| `TZ` | 服务器时区，如 `Asia/Shanghai`。 |

### 数据库 DSN

同一套代码只需修改 `DSN` 即可切换数据库：

```env
# PostgreSQL
DSN="postgres://daqing:passwd@127.0.0.1:5432/airway"

# SQLite（文件或内存）——纯 Go 驱动，无需 CGO
DSN="sqlite://./tmp/airway.db"
DSN="sqlite://:memory:"

# MySQL 8（URL 或原生驱动格式）
DSN="mysql://root:passwd@127.0.0.1:3306/airway?charset=utf8mb4"
DSN="root:passwd@tcp(127.0.0.1:3306)/airway?charset=utf8mb4&parseTime=true"
```

说明：

- 基础 CRUD 在 PostgreSQL、MySQL 8 与 SQLite 之间可移植。
- 部分高级 SQL Builder 助手（ARRAY、JSONB、少量 lateral/window 表达式）仍面向 PostgreSQL。
- SQLite 使用纯 Go 的 `modernc.org/sqlite` 驱动。

## HTTP 端点

注册于 `config/routes.go`：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/` | 首页（HTML，由 templ 视图渲染）。 |
| GET | `/ui` | airway-ui 组件展示页（交互式 island）。 |
| GET | `/openapi.json` | 实时 OpenAPI 3.2 文档（见 [API 文档](#api-文档openapi)）。 |
| GET | `/health` | 健康检查。 |
| GET | `/ws` | WebSocket 连接。 |
| POST | `/ws/publish` | 向已连接客户端发布消息（表单字段 `message`）。 |
| POST | `/api/v1/storage` | 上传文件（multipart `file`，可选 `dir`）。 |
| GET | `/api/v1/storage/*key` | 下载文件。 |
| DELETE | `/api/v1/storage/*key` | 删除文件。 |
| GET | `/api/v1/ui-demo/items` | `/ui` 页面 TanStack island 的演示数据。 |

设置了 `URL_PREFIX` 后，公开路由——首页、WebSocket 与 API——只会在该前缀下服务，不再出现在根路径。健康检查仍保持在无前缀的根路径可达，便于负载均衡探针直接访问 `/health`。以 `URL_PREFIX="/airway"` 为例：

```bash
curl http://127.0.0.1:1900/airway/health
curl http://127.0.0.1:1900/health          # 仍然可达（探针）
curl -F "file=@report.pdf" http://127.0.0.1:1900/airway/api/v1/storage
```

客户端——包括 WebSocket 连接——必须带上前缀（`ws://host:1900/airway/ws`）。API 返回的本地存储 URL 会带上前缀；云/CDN URL 不受影响。

## HTML 视图（templ）

页面是 `app/views/` 下的 [templ](https://templ.guide/) 模板，每个 API 模块一个目录——目录名去掉 `_api` 后缀，因此 `home_api` 渲染 `app/views/home/index.templ`。每个目录是独立的包；共享的文档外壳位于 `app/views/layouts/base.templ`。action 通过 `lib/render` 的 HTML 助手返回组件：

```go
// app/api/home_api/index_action.go
render.HTML(c, home.Index())
```

修改任何 `.templ` 文件后，重新生成 Go 代码并保持提交：

```bash
go generate ./...   # 或：just generate，或：airway templates:compile
```

生成的 `*_templ.go` 文件已提交，因此构建与测试不需要 templ CLI。

### 交互式 islands

页面保持服务端渲染；交互区域是 islands——Preact TSX 组件挂载到带服务端 props 的 `data-island` 节点上：

```go
// 在 .templ 视图中（见 app/views/home 的实际示例）
@assets.Island("counter", map[string]any{"start": 3})
```

组件位于 `app/assets/js/islands/counter.tsx`（默认导出即可；文件路径就是 island 名称），会被自动打包——无需手工注册。本地开发时 bundle 在内存中重建、浏览器自动刷新；生产环境中它内嵌在二进制里，通过防缓存 URL 提供。页面完全不依赖 JavaScript 也能完整渲染：挂载点初始为空。

**airway-ui** 是 islands 所基于的内置组件库：按钮、输入框、表单（react-hook-form）、表格（TanStack Table）、模态框、toast、标签页，以及与 `lib/render` JSON 信封对齐的请求层——视觉层自研，逻辑层来自 preact/compat 生态。启动服务后可在 [`/ui`](http://127.0.0.1:1900/ui) 实时查看全部组件。

## 前端策略

状态：已端到端落地——见[前端指南](frontend.md)。整条管线随 CLI 交付
（`js:add`/`js:install`/`js:build`、`generate island`/`scaffold`），随项目模板交付，也随本仓库自身交付（首页计数器与 `/ui` 组件展示页都是 islands）。

前端代码与 Go 代码同仓库，获得媲美现代 UI 框架的组件化工作流，并且**不引入 Node.js 工具链**：

- **templ 渲染骨架**——页面结构、SEO、首屏。
- **交互区域是 islands**——`app/assets/js/` 下的 Preact TSX 组件，挂载到标记 `data-island` 的元素上；初始数据序列化在挂载点旁。
- **esbuild 作为 Go 库内嵌**——CLI 直接链接 `github.com/evanw/esbuild/pkg/api`：`airway js:build` 编译 TS/TSX，开发服务器从内存提供重建后的 bundle，生产 bundle 经 `go:embed` 内嵌进单个 Go 二进制。
- **自研 `airway-ui` 组件库**，构建在 `preact/compat` 之上——重逻辑的 React 生态库（TanStack Table/Query/Form、React Hook Form）保持可用，视觉层保持自研。

已否决的备选方案：

- **htmx + Alpine.js（HTML over the wire）**——做渐进增强没问题，但没有基于组件的响应式编程模型；交互能力上限远不及真正的组件库。
- **Node 前端子项目（Vue、React/Next.js、Svelte 等）经 `go:embed` 内嵌**——把完整的 Node 工具链拖进仓库；真到那一步，前后端分离才是更诚实的选择。
- **LiveView 式服务端驱动 UI**——对 Go 框架实际价值有限；当应用确实需要重度前端工程时，把前端拆出去独立开发才是正解。

逃生舱：当应用超出 islands 的能力范围（复杂 SPA、富文本编辑器），应把前端拆成独立项目（Vue、React、Svelte 等），Airway 退化为纯 JSON API。

## API 文档（OpenAPI）

二进制提供的每个 API 路由都会被记录为 **OpenAPI 3.2** 文档——无需任何注解。`airway openapi:generate` 扫描路由、从 handler 名推导 operationId，写出 `./openapi.json`（构建产物，已 git-ignore——路由变化后重新生成即可）。同一份文档实时服务于 [`/openapi.json`](http://127.0.0.1:1900/openapi.json)。

各模块通过 `openapi.go` 文件用 `lib/openapi`（`openapi.Get(...)` 等）在代码里富化文档，声明操作及其从 Go 类型推导的请求/响应 schema；文档级元数据位于 `app/api/openapi_api/doc.go`。详见 [OpenAPI 指南](openapi.md)。

## Admin 后台

`airway admin:generate` 读取 TOML 表规格（`config/admin.toml`），生成一个完整、可上生产的管理后台——它是真实的、归用户所有的 Go 代码：cookie 会话认证、角色、审计日志、CSV 导出、服务端分页列表，并为每个声明字段（含 datetime、enum、references、attachment）生成类型感知的表单与筛选。

```bash
airway admin:generate                    # 或：admin:generate --force=table1,table2
airway admin:root admin 's3cret'         # 创建管理员账号
airway admin:member editor 's3cret'      # 创建非管理员后台账号（--role=editor|viewer）
airway server                            # 在 /admin/login 登录
```

详见 [Admin 后台指南](../../ADMIN.zh-CN.md)（[English](../../ADMIN.md)）。

## 桌面应用

`airway desktop:init` 把项目导出为 **Wails v3** 桌面目标（`./desktop`）：同一套 Web 技术栈（templ 视图、islands、JSON API、WebSocket）在桌面进程内的本地回环端口上运行，原生 WebView 窗口加载该地址——服务端渲染页面、cookie 会话、重定向和 WebSocket 的行为与 Web 上完全一致，应用代码零改动。SQL 迁移内嵌并在启动时自动应用；打包覆盖 macOS（.app）、Windows（NSIS）与 Linux（deb/rpm/AppImage）。

```bash
airway desktop:init       # 生成 ./desktop；重跑可重新同步迁移/插件
```

详见[桌面应用指南](desktop.md)（[English](../desktop.md)）与调研记录 [WAILS.md](../../WAILS.md)。

## 静态展示站点（SSG）

Airway 同时也是面向展示型网站的静态站点生成器——公司官网、产品落地页、作品集。`airway ssg:new` 脚手架一个站点项目，页面在 `ssg.go` 中以 Go 代码声明，外观来自可替换的**主题 module**；`airway ssg:build` 导出纯 HTML 目录，可部署到任何静态托管；`airway ssg:serve` 本地预览。主题就是普通的 Go module（templ 组件 + 内嵌资源）：`airway theme:new` 生成新主题，`airway theme:install` 把主题装进站点。框架自带 corporate 参考主题。

```bash
airway ssg:new mysite       # 然后：airway ssg:build / airway ssg:serve
```

设计记录见 [SSG.zh-CN.md](../../SSG.zh-CN.md)（[English](../../SSG.md)），使用指南见 [docs/zh-CN/ssg.md](ssg.md)（[English](../ssg.md)）。

## CLI

Airway CLI 是单个 `airway` 二进制（`go install github.com/daqing/airway@latest` 安装）。在项目内它会检测宿主应用，并把所有项目级命令透明地经 `go run .` 重新执行（stderr 显示 `proxying to project binary` 提示），因此插件、REPL 模型与 Go 代码迁移总是来自项目自己的二进制。命令会自动加载项目根目录的 `.env`。

### 项目与服务

```bash
airway new <module-path | /path>            # 生成新项目骨架
airway server                               # 启动 HTTP 服务
airway generate api admin                   # 在 app/api/ 下新建 API 命名空间
airway generate action admin show           # 在已有 API 模块中新建 action
airway generate model post                  # 在 app/models/ 中新建模型
airway generate service post title:string   # 在 app/services/ 中生成 CRUD service
airway generate island chart                # 交互式 island 组件
airway generate scaffold post title:string  # 全套 CRUD：模型+迁移+API+页面+island
airway generate migration create_posts      # 在 db/migrate/ 中生成 .up.sql/.down.sql 对
airway repl                                 # 交互式 repo REPL（项目内自动代理到 go run .）
airway version                              # 打印版本（亦支持 -v、--version）
```

### 数据库

```bash
airway db:create                            # 创建数据库
airway db:drop                              # 删除数据库
airway db:migrate [version]                 # 应用迁移
airway db:rollback [step]                   # 回滚迁移
airway db:status                            # 迁移状态
airway schema:dump                          # 写出 db/schema.json
airway schema:show                          # 打印 db/schema.json
```

### 前端

```bash
airway js:add <pkg>[@version]               # 添加前端 npm 依赖（无需 Node）
airway js:install                           # 将 js.pkg.json 依赖安装到 app/assets/js/vendor/
airway js:build                             # 将 app/assets/js 打包到 app/assets/dist（esbuild）
airway templates:compile                    # 重新生成 templ 视图（`go generate ./...` 的简写）
```

### API 文档

```bash
airway openapi:generate [--out path]        # 写出 OpenAPI 3.2 文档（默认 ./openapi.json）
```

### Admin 后台

```bash
airway admin:generate [config/admin.toml]   # 从 TOML 表规格生成后台
airway admin:root <username> <password>     # 创建管理员账号（role admin）
airway admin:member <username> <password> [--role=editor|viewer]
                                            # 创建非管理员后台账号
```

### 桌面应用

```bash
airway desktop:init [--force]               # 在 ./desktop 生成 Wails v3 桌面目标
```

### 静态站点与主题

```bash
airway ssg:new [--local[=path]] <name>      # 脚手架静态展示站点项目
airway ssg:build [--out dist]               # 将 ssg.go 定义的站点导出为静态 HTML
airway ssg:serve [--addr 127.0.0.1:3000]    # 本地服务预览站点
airway theme:new [--local[=path]] <name>    # 脚手架新的站点主题 module
airway theme:install <module | /path>       # 把站点主题接入宿主项目
```

### 静态导出（应用页面）

```bash
airway static:build [--out dist]            # 将 export.go 注册的页面 + 前端产物导出为静态 HTML
airway static:serve [--addr 127.0.0.1:3000] # 本地服务预览静态页面
```

### 插件

```bash
airway plugin:new <module-path | /path>     # 脚手架新插件 module
airway plugin:list                          # 已注册插件及其挂载路径
airway plugin:install <module>              # 启用插件 + 安装其 SQL 迁移与 deps/
airway plugin:lint                          # 检查当前插件项目的旧版布局问题
```

### 文件

```bash
airway upload [key] /path/to/file           # 经配置的存储上传
```

数据库命令读取 `DSN`/`AIRWAY_DSN`。完整内容见
[CLI 指南](cli.md)。

## Repository API（`lib/repo`）

### 模型

模型是带 `db` tag 和 `TableName()` 方法的结构体。关联通过 `Relations()` 声明。

```go
import "github.com/daqing/airway/lib/repo"

type User struct {
	ID      int64    `db:"id"`
	Name    string   `db:"name"`
	Email   string   `db:"email"`
	Profile *Profile // belongs_to
	Posts   []*Post  // has_many
}

func (User) TableName() string { return "users" }

func (User) Relations() map[string]repo.Relation {
	return map[string]repo.Relation{
		"Profile": repo.HasOne(Profile{}, "UserID"),
		"Posts":   repo.HasMany(Post{}, "UserID"),
	}
}

type Post struct {
	ID     int64  `db:"id"`
	UserID int64  `db:"user_id"`
	Title  string `db:"title"`
	Author *User  // belongs_to
}

func (Post) TableName() string { return "posts" }

func (Post) Relations() map[string]repo.Relation {
	return map[string]repo.Relation{
		"Author": repo.NewBelongsTo(User{}, "UserID"),
	}
}
```

### CRUD

所有助手使用启动时配置的数据库（`repo.SetupDB`），并以模型类型作为类型参数：

```go
// 创建
user, err := repo.CreateFrom[User](sql.H{"name": "John", "email": "john@example.com"})

// 读取
user, err := repo.FindByID[User](1)
user, err := repo.FindOneBy[User](sql.H{"email": "john@example.com"})
users, err := repo.FindBy[User](sql.H{"active": true})
users, err := repo.FindAll[User]()

// 更新
err := repo.UpdateByID[User](1, sql.H{"name": "Jane"})
err := repo.UpdateWhere[User](sql.H{"status": "inactive"}, sql.Eq("last_login_at", nil))

// 删除
err := repo.DeleteByID[User](1)
err := repo.DeleteWhere[User](sql.H{"status": "banned"})

// 存在性 / 计数
ok, err := repo.ExistsWhere[User](sql.H{"email": "john@example.com"})
n, err := repo.CountWhere[User](sql.H{"active": true})
n, err := repo.CountEvery[User]()
```

### 事务

`repo.WithTx` 在单条事务绑定连接上运行回调。回调收到一个 `*repo.Tx`，其助手方法都在该事务上执行；泛型助手通过 `*With` 变体传入 `tx.Executor()`：

```go
users := sql.TableOf("users")

err := repo.WithTx(db, func(tx *repo.Tx) error {
	if _, err := repo.InsertWith[User](tx.Executor(), sql.Insert(sql.H{"name": "John"}).IntoTable(users)); err != nil {
		return err
	}

	n, err := tx.Count(sql.SelectColumns("count(*)").FromTable(users))
	if err != nil {
		return err
	}

	return nil // 提交；返回非 nil 即回滚
})
```

用 `tx.Raw()` 可以拿到 `*sql.Tx` 执行手写 SQL。没有公开的
`Commit`/`Rollback`——结果跟随回调的返回值，`repo.WithTxContext` 支持传入
context。`JoinQuery`/`Preloader` 仅限连接池，不能在事务内运行。

乐观锁配方：用 `UpdateAffected` 加版本检查，保证并发写只有最新者胜出：

```go
affected, err := tx.UpdateAffected(
	sql.UpdateTable(users).
		Set(sql.H{"name": "Jane", "version": sql.Expr("version + 1")}).
		Where(sql.AllOf(sql.Eq("id", 1), sql.Eq("version", currentVersion))),
)
if err != nil {
	return err
}
if affected == 0 {
	return errors.New("stale record")
}
```

事务以 PostgreSQL 为主要目标。MySQL 的插入路径会在同一连接上补发一次查询，在事务内属于尽力而为。

### 预加载（eager loading）

Preload 用几条查询替代 N+1 循环：

```go
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Profile", "Posts").Exec(&users)

// 嵌套与条件预加载
err := repo.Preload("Posts").ThenPreload("Comments").Exec(&users)
err := repo.PreloadCond("Posts", sql.AllOf(
	sql.Eq("published", true),
	sql.Gte("created_at", "2024-01-01"),
)).Exec(&users)
```

### 联表

```go
results, err := repo.Join(User{}).LeftJoins("Profile").Find()
results, err := repo.Join(User{}).
	Joins("Posts", sql.Gt("posts.views", 100)).
	Where(sql.Eq("users.active", true)).
	OrderBy("users.name ASC").
	Page(1, 20).
	Find()

count, err := repo.Join(User{}).Joins("Posts").Count()

var users []*User
err := repo.Join(User{}).LeftJoins("Profile").FindInto(&users)
```

### Rails 对照

| Rails ActiveRecord | Airway |
| --- | --- |
| `User.find(id)` | `repo.FindByID[User](id)` |
| `User.find_by(email: e)` | `repo.FindOneBy[User](sql.H{"email": e})` |
| `User.where(active: true)` | `repo.FindBy[User](sql.H{"active": true})` |
| `User.all` | `repo.FindAll[User]()` |
| `User.create(attrs)` | `repo.CreateFrom[User](attrs)` |
| `User.update(id, attrs)` | `repo.UpdateByID[User](id, attrs)` |
| `User.delete(id)` | `repo.DeleteByID[User](id)` |
| `User.joins(:profile)` | `repo.Join(User{}).Joins("Profile")` |
| `User.includes(:posts)` | `repo.Preload("Posts").Exec(&users)` |
| `User.count` | `repo.CountEvery[User]()` |
| `User.where(active: true).count` | `repo.CountWhere[User](sql.H{"active": true})` |

## Repo REPL

直接对已配置的数据库操作 `lib/repo`：

```bash
go run . repl                              # 使用已配置的 DSN
go run . repl --driver sqlite --dsn ./tmp/airway.db
```

REPL 只能看到它所在二进制编译进的模型（经 `lib/replreg` 注册）。在项目内，`airway repl` 会自动代理到 `go run . repl`，因此你项目的模型会出现；项目外只能看到框架内置模型。

命令：`help`、`driver`、`tables`、`exit`。输入 Go 表达式即可求值——builder 会打印编译后的 SQL，`repo.*` 调用直接作用于数据库：

```text
repo.FindOne("posts", sql.Eq("id", 1))
repo.Find[models.Post](pg.Select("id").Where(sql.Eq("id", 1)))
repo.Insert[models.Post](pg.H{"id": 1234, "title": "hello"})
repo.Update("posts", pg.H{"published": false}, sql.Eq("id", 1))
repo.Delete("posts", sql.Eq("id", 1))
pg.Select("*").From("posts").Where(sql.Eq("id", 1))
```

可用命名空间：`repo`、`sql`、`pg`、`mysql`、`sqlite`、`models`。
`repo.Find`/`FindOne`/`Count`/`Exists` 接受构建好的语句或 `表 + 条件`；类型化调用支持匿名结构体与应用模型。全表更新/删除需要显式传入 `true` 作为最后一个参数。

## 文件存储

`lib/storage` 是本地目录或云后端（S3 / R2 / COS）之上的统一层。通过 `STORAGE_DRIVER` 配置后端：

```env
# 本地
STORAGE_DRIVER="local"
STORAGE_ROOT="./data/storage"

# 云（S3 兼容）：可选 s3 / r2 / cos
STORAGE_DRIVER="s3"
STORAGE_REGION="us-east-1"
STORAGE_BUCKET="my-bucket"
STORAGE_ACCESS_KEY="..."
STORAGE_SECRET_KEY="..."
# STORAGE_PUBLIC_URL="https://cdn.example.com"   # 可选 CDN 基址
```

在代码中经当前后端使用：

```go
store := storage.Current() // 启动时由 storage.Setup 安装

err := store.Put(ctx, "docs/report.pdf", storage.Object{
	Reader:      file,
	Size:        size,
	ContentType: "application/pdf",
})
rc, err := store.Get(ctx, "docs/report.pdf") // 用完关闭
ok, err := store.Exists(ctx, "docs/report.pdf")
url, err := store.URL(ctx, "docs/report.pdf", 24*time.Hour)
err := store.Delete(ctx, "docs/report.pdf")
```

`URL()` 对本地后端返回应用自身的下载路径，对云后端返回 CDN 或预签名 URL。应用在同一层之上暴露 REST API：

```bash
# 上传（multipart 字段 "file"，可选 "dir"）
curl -F "file=@report.pdf" -F "dir=docs" http://127.0.0.1:1900/api/v1/storage
# => {"key":"docs/202609/ab12cd....pdf","url":"/api/v1/storage/docs/202609/ab12cd....pdf","size":12345}

# 下载 / 删除
curl -O http://127.0.0.1:1900/api/v1/storage/docs/202609/ab12cd....pdf
curl -X DELETE http://127.0.0.1:1900/api/v1/storage/docs/202609/ab12cd....pdf
```

完整内容见[文件存储指南](storage.md)。

## 部署

二进制是自包含的。构建纯 Go 镜像（无需 CGO 工具链）：

```bash
just docker        # 或：docker build -t airway .
docker run -p 1900:1900 -e AIRWAY_ENV=production -e DSN="sqlite:///app/tmp/airway.db" airway
```

Dockerfile 从 daocloud.io 镜像拉取基础镜像，并把 Go module 代理指向
goproxy.cn，因此在默认源缓慢或不可达的网络环境下也能开箱构建；不需要镜像
的话移除即可。

部署前后运行迁移：

```bash
./airway db:migrate
```

compose 示例见 [docs/docker-compose.yml.example](../docker-compose.yml.example)。如需在反向代理后以路径前缀服务，设置 `URL_PREFIX`（如 `/airway`）——见 [HTTP 端点](#http-端点)。
