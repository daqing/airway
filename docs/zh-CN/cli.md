# Airway CLI 脚手架使用说明

Airway 内置脚手架 CLI，可以全局安装：

```bash
go install github.com/daqing/airway@latest
```

安装后即得到 `airway` 命令。命令执行时会优先自动加载当前项目根目录下的 `.env` 文件。
在项目内部，全局安装的 `airway` 会识别宿主应用（go.mod require 了
`github.com/daqing/airway` 且存在 main.go），并自动把所有项目级命令透明地转为
`go run .` 重新执行——Plugin、REPL 模型、Go 代码迁移始终来自项目自己的二进制。
只需记住 `airway <命令>` 这一种用法，在哪里都成立。代理时会在 stderr 打印一行
`proxying to project binary: go run . ...` 提示。只有当全局安装的 CLI 版本与项目
go.mod 中 pin 的 airway 版本一致时代理才会执行（若通过 `replace` 指向本地 airway
检出，则改与该目录的 `VERSION` 文件比对）；版本不一致时命令直接报错退出，避免
静默运行项目二进制里过期的 CLI 逻辑。`new`、`version`、`help`
始终本地执行；项目之外的命令行为不变；直接使用 `go run . <命令>` 与代理等价。

旧形式 `airway cli <命令>` 仍作为兼容别名可用。

## 命令总览

```bash
airway new <module-path | directory>                    # 生成新项目骨架
airway server                                           # 启动 HTTP 服务
airway db:create
airway db:drop
airway db:migrate [version]
airway db:rollback [step]
airway db:status
airway plugin:new <module-path>                           # 生成新的 Plugin 模块骨架
airway plugin:list
airway plugin:install <module>
airway generate [action|api|model|migration|service|island|scaffold|cmd] [params]
airway schema:dump
airway schema:show
airway openapi:generate [--out path]                     # 生成 OpenAPI 3.2 文档（默认 ./openapi.json）
airway admin:generate [config/admin.toml]              # 从 TOML 表配置生成完整 Admin 后台
airway admin:root <username> <password>               # 创建管理员账号（role admin）
airway admin:member <username> <password> [--role=r]  # 创建普通账号（editor|viewer）
airway desktop:init [--force]                           # 在 ./desktop 生成 Wails v3 桌面目标（见 docs/zh-CN/desktop.md）
airway templates:compile                                 # 重新编译 templ 视图（等价于 `go generate ./...`）
airway upload /path/to/file
airway repl
airway version                                           # 或 -v / --version；打印 VERSION 文件内容
```

不带参数运行 `airway` 会打印用法说明。

## 创建新项目

```bash
airway new myapp                    # 目录名：myapp
airway new github.com/me/myapp      # module 路径；目录取路径最后一段
airway new /path/to/myapp           # 本地路径；在该位置创建项目，module 为 myapp
```

`airway new` 会以框架仓库的 `app/` 骨架为模板生成一个新项目，自动从
`.env.example` 生成 `.env`，执行 `go mod tidy`，安装前端依赖（`airway
js:install`），并打印后续步骤：

```bash
cd myapp
# 编辑 .env —— 配置 DSN 和 PORT
airway db:create
airway db:migrate
go run .                # 启动服务器（等同于 go run . server）
```

新项目的二进制不带参数（或带 `server`）时启动 HTTP 服务；带其他参数时派发给
内置 CLI。

## 启动服务器

```bash
airway server        # 或者在源码目录中：go run . server
```

框架仓库根目录的 `main.go` 不再默认启动 HTTP 服务——开发框架本身时请使用
`go run . server`。Docker 镜像已经以 `server` 参数启动。

环境变量始终优先于 `.env`：`.env` 只为进程环境未设置的键提供回退值，因此
`PORT=1988 go run . server` 即使在 `.env` 定义了 `PORT` 或 `AIRWAY_PORT` 时
也会监听 1988。只要 `.env` 提供了 `AIRWAY_ENV`，环境中不带它也能启动服务。

## 上传文件

使用 `.env` 中的 storage 配置上传本地文件：

```bash
airway upload /tmp/foo.png
```

源文件路径会转换为相对于存储根目录的 key。上例的 key 是 `tmp/foo.png`，
命令输出中显示为 `/tmp/foo.png`。文件大小来自文件信息；Content-Type 优先
根据扩展名确定，无法确定时再检测文件内容。

如果需要明确指定 storage key，可以把 key 放在本地文件路径之前：

```bash
airway upload images/foo.png /tmp/foo.png
```

## 代码生成命令

生成器会读取当前目录 `go.mod` 中的 module 路径，因此生成的 service/cmd 代码
import 的是项目自身的 `app/models`、`app/services` 包，而不是硬编码的框架路径。

### 生成 API 模块

```bash
airway generate api admin
```

会创建：

- `app/api/admin_api/routes.go`
- `app/api/admin_api/index_action.go`

适合在你准备新增一个 API 命名空间时使用。

### 在已有 API 模块里生成 action

```bash
airway generate action admin show
```

会创建：

- `app/api/admin_api/show_action.go`

适合在现有 API 目录下继续新增接口处理函数。

### 生成 model

```bash
airway generate model post
```

会创建：

- `app/models/post.go`

生成内容默认包含：

- `ID`、`CreatedAt`、`UpdatedAt`
- `TableName()`
- 供 REPL 使用的 `registerREPLModel`

### 生成 service

```bash
airway generate service post title:string published:bool
```

会创建：

- `app/services/post.go`

默认生成的方法包括：

- `FindPost`
- `CreatePost`
- `UpdatePost`
- `DeletePost`

字段参数格式为 `name:type`。

### 生成命令辅助代码

```bash
airway generate cmd post title published
```

会创建：

- `cmd/post.go`

这个生成器适合给项目补充围绕 service 的命令行辅助函数。

### 生成迁移文件

```bash
airway generate migration create_posts
```

会在 `db/migrate/` 下生成一对带时间戳的 SQL 文件：

- `<时间戳>_create_posts.up.sql` —— 正向迁移
- `<时间戳>_create_posts.down.sql` —— 回滚迁移

两个文件里带有注释掉的 `CREATE TABLE` / `DROP TABLE` 示例，编辑成你需要的
表结构即可。

旧的 Go DSL 迁移机制（`lib/migrate/schema` 的 `schema.RegisterChange`）仍然保留，
但 DSL 迁移只在编译进执行迁移的二进制时生效。CLI 在 `./db/migrate` 下发现
时间戳命名的 `.go` 迁移文件时会打印警告，提醒这一点。

## 数据库迁移命令

### 执行全部待运行迁移

```bash
airway db:migrate
```

### 迁移到指定版本

```bash
airway db:migrate 20260327120000
```

### 回滚最近一次迁移

```bash
airway db:rollback
```

### 按步数回滚

```bash
airway db:rollback 3
```

### 查看迁移状态

```bash
airway db:status
```

迁移相关命令读取数据库连接串的顺序为：

1. `AIRWAY_DSN`
2. `DSN`
3. 兼容旧项目时依次回退到 `AIRWAY_DB_DSN`、`AIRWAY_PG`

在本地开发场景下，CLI 会自动加载项目根目录的 `.env` 文件，因此通常直接把 `DSN` 写在 `.env` 里即可。
迁移命令会复用 Airway 当前 DSN 所对应的数据库类型，因此支持项目当前支持的 PostgreSQL、MySQL 和 SQLite。

## Plugin 命令

生成一个新的 Plugin 模块骨架（独立的 Go module；见
[Plugin 扩展机制](plugin.md)）：

```bash
airway plugin:new im                              # 目录：im，Plugin 名称：im
airway plugin:new github.com/me/airway-im-plugin  # 名称从路径最后一段推导
```

与 `plugin:list` 不同，`plugin:new` 不涉及项目二进制——它只写文件。

Plugin 是通过 `plugins.go` 中的 blank import 启用的可选功能模块（见
[Plugin 扩展机制](plugin.md)）：

```bash
airway plugin:list               # 列出已注册的 Plugin 及挂载路径
airway plugin:install <module>   # 安装 Plugin 的 SQL 迁移、host/ 目录树和 deps/ 目录
```

Plugin 在编译期注册，所以在项目内全局安装的 `airway` 会自动把这些命令转为
`go run .` 执行（只有项目自己的二进制能看到已启用的 Plugin）；直接执行
`go run . plugin:list` 效果相同。

`plugin:install` 的参数是 Plugin 的模块路径（如 `github.com/daqing/airway-im-plugin`)，
插件名从路径最后一段推导（与 `plugin:new` 相同）。它会为复制的迁移文件分配新的时间戳，并跳过已安装的文件；复制后它们就是
普通迁移，由 `db:migrate` / `db:rollback` / `db:status` 统一管理。

## OpenAPI 命令

```bash
airway openapi:generate                    # 生成 ./openapi.json（OpenAPI 3.2）
airway openapi:generate --out docs/api.json
```

为 Gin 引擎上注册的每一条路由（含 Plugin 挂载的路由）生成确定性的 OpenAPI
3.2 文档；输出经过排序，路由变化后重新生成不会产生无意义的 diff，该文件是
本地构建产物（已被 git 忽略）。运行中的服务同时在
`GET /openapi.json` 上提供实时文档（配置了 `URL_PREFIX` 时挂载在前缀之下），
`servers` 由请求 Host 推导。

未声明的路由按框架默认 JSON 信封生成文档；模块通过 `openapi.go` 文件补充
请求/响应 schema——声明 API 与客户端生成方案（Vue 3 / React 用
openapi-typescript 或 orval，SwiftUI 用 swift-openapi-generator）见
[OpenAPI 指南](openapi.md)。

## REPL

```bash
go run . repl
```

REPL 只能看到编译进当前二进制、通过 `github.com/daqing/airway/lib/replreg`
注册的模型——项目模型的 init 通过 `app/models` 的 `registerREPLModel` 注册
（该函数委托给 `lib/replreg`）。在项目内，全局安装的 `airway repl` 会自动代理为
`go run . repl`；在项目之外则只能看到框架自带的模型。

## 实战示例

下面用一个最小例子，演示如何从零生成一个 `posts` 功能模块。

### 第 1 步：生成 migration

```bash
airway generate migration create_posts
```

然后编辑 `db/migrate/` 下面新生成的 `.up.sql` 文件，写入表结构。

例如：

```sql
CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

执行迁移：

```bash
airway db:migrate
```

### 第 2 步：生成 model

```bash
airway generate model post
```

这会创建 `app/models/post.go`。

生成完成后，通常还需要把真实字段补进去，例如：

```go
type Post struct {
	ID        sql.IdType `db:"id" json:"id"`
	Title     string     `db:"title" json:"title"`
	Published bool       `db:"published" json:"published"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}
```

### 第 3 步：生成 service

```bash
airway generate service post title:string published:bool
```

这会创建 `app/services/post.go`，里面带有基础 CRUD 方法。

### 第 4 步：生成 API 模块

```bash
airway generate api post
airway generate action post create
airway generate action post show
```

这会生成：

- `app/api/post_api/routes.go`
- `app/api/post_api/index_action.go`
- `app/api/post_api/create_action.go`
- `app/api/post_api/show_action.go`

### 第 5 步：把 API routes 接到总路由

打开 [config/routes.go](/Users/daqing/mzevo/open-source/airway/config/routes.go)，先引入生成出来的包：

```go
import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/post_api"
	"github.com/daqing/airway/app/api/health_api"
	"github.com/daqing/airway/app/websocket"
)
```

然后在 `apiGroupRoutes` 里注册：

```go
func apiGroupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		post_api.Routes(v1)
	}
}
```

按默认生成的 `routes.go`，你会得到类似这样的接口地址：

```text
GET /api/v1/post/index
```

### 第 6 步：补 action 里的业务逻辑

例如 `create_action.go` 现在还只是脚手架，你还需要继续补：

- 请求参数定义
- 调用 `services.CreatePost(...)`
- 使用 `render.OK(...)` 或 `render.Error(...)` 返回结果

### 第 7 步：启动项目

```bash
just
```

或者：

```bash
go run . server
```

到这里，你已经把下面这几层骨架都搭起来了：

- 数据库 migration
- model
- service
- API handler
- 路由注册

## 补充说明

- 生成器不会覆盖已有文件；如果目标文件已经存在，命令会直接返回 `file already exists`。
- `generate api` 只负责生成 API 目录和文件，你仍然需要手动把生成的 `Routes(...)` 接入路由配置。
- `generate service` 默认假设你的项目里有 `app/services` 包。
- 生成出来的代码是脚手架起点，通常还需要继续补业务逻辑。

## 生成 Admin 管理后台

`airway admin:generate` 可以把一份 TOML 配置文件变成完整的后台管理系统：
带 cookie 会话的登录、每个资源一张卡片的仪表盘、侧边栏导航，以及每张表
的完整 CRUD（JSON API + 类型感知的 CRUD island）。配置文件默认是
`config/admin.toml`（也可以把其他路径作为唯一参数传入）。每个顶层表就是
一个资源，键名为**单数**形式；每个字段把列名映射为一个类型：

```toml
[category]
name = "string"
sort_order = "integer"
parent_id = "references:category"   # 自引用（显式指定目标表）

[post]
title = "string"
body = "text"
views = "integer"
score = "float"
published = "boolean"
published_at = "datetime"
status = "enum:draft,published,archived"
cover = "attachment"
category_id = "references"          # 目标表根据 _id 后缀推断
```

支持的字段类型：`string`、`text`、`integer`、`float`、`boolean`、
`datetime`、`enum:a,b,c`、`references[:table]` 和 `attachment`。每张表
都会自动带上 `id`、`created_at`、`updated_at`，不要在配置里声明。

运行生成器，然后执行标准的后续步骤：

```bash
airway admin:generate
airway templates:compile                  # 编译 .templ 视图
airway js:build                           # 打包 CRUD island
airway db:migrate                         # 建表
airway admin:root admin      # 创建第一个管理员账号
airway server                             # 访问 /admin
```

生成的内容包括：

- `app/models/` 里每张表一个 model（带 references 字段的表同时生成
  `Relations()`），以及认证用的 `admin_user.go` / `admin_session.go`。
- `app/api/admin_api/`：每张表一个自注册的 resource 文件（CRUD 动作和
  OpenAPI 声明），外加共享文件（registry、路由、登录/登出、通过
  `storage.Current()` 的附件上传）。
- `app/views/admin/`：带侧边栏的 Admin 布局、仪表盘、登录页，以及每张
  表一个承载 CRUD island 的页面。
- 每次运行一对 migration（首次运行包含认证表，之后按外键依赖顺序创建
  新表）。

路由挂载在 `/admin`（页面）和 `/api/v1/admin`（JSON API）下，都受
`AdminAuth` 中间件保护：没有有效会话时页面重定向到 `/admin/login`，
API 调用返回 401。管理员账号用 `airway admin:root <username> <password>`
创建，普通账号（editor/viewer）用 `airway admin:member` 创建——密码经
bcrypt 哈希，会话保存在服务端的 `admin_sessions` 表中。

角色与安全加固开箱即用：

- **角色**——`admin`（全部权限，可查看审计日志）、`editor`（默认；读写）、
  `viewer`（只读：写操作和上传返回 403）。写路由挂在 `AdminRequireWrite`
  中间件之后。
- **CSRF**——登录表单携带 double-submit token，必须与
  `airway_admin_csrf` cookie 匹配。
- **限速**——同一 IP 和用户名组合五次登录失败将被锁定十五分钟
  （`lib/ratelimit`）。
- **审计日志**——每次增删改都会记录操作账号；admin 可在
  `/admin/audit-log` 查看。
- **服务端列表**——列表 API 支持 `page`、`page_size`、`q`（文本搜索）、
  `sort`/`order`（白名单列）以及对任意声明字段的精确过滤
  （`?status=published`）。CRUD island 内置搜索框、分页和导出 CSV 按钮。
- **软删除**——在表里声明 `deleted_at = "datetime"` 后，删除操作改为写入
  该列，读取时自动过滤。
- **显示标签**——可选的 `[table.meta]` 段可覆盖侧边栏标签（`label`）和
  字段标签（`labels`），面板可以使用 TOML 里的任何语言。

往 TOML 里加表之后重新运行 `admin:generate` 是纯增量操作：已有文件不会被
改写，registry 会自动发现新资源（侧边栏和仪表盘也随之更新）。要重新生成
已有表的代码——例如修改了 TOML 或想应用新的生成器能力——传入
`--force`（或 `--force=table1,table2`）：它会重写该表的生成文件并丢弃手改
内容，但绝不改动 migration，schema 变更仍需手写 migration。从 TOML
里删除表不会删除已生成的代码。
