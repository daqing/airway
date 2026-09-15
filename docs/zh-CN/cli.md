# Airway CLI 脚手架使用说明

Airway 内置脚手架 CLI，可以全局安装：

```bash
go install github.com/daqing/airway@latest
```

安装后即得到 `airway` 命令。命令执行时会优先自动加载当前项目根目录下的 `.env` 文件。
在项目（或框架仓库）内部，同样的命令也可以用 `go run . <命令>` 的方式执行——其中
`repl`、`plugin:install` 等命令*建议*用这种方式运行，因为它们只能看到编译进当前
二进制的模型和 Plugin（详见下文）。

旧形式 `airway cli <命令>` 仍作为兼容别名可用。

## 命令总览

```bash
airway new <module-path>                                # 生成新项目骨架
airway server                                           # 启动 HTTP 服务
airway db:create
airway db:drop
airway db:migrate [version]
airway db:rollback [step]
airway db:status
airway plugin:new <module-path>                           # 生成新的 Plugin 模块骨架
airway plugin:list
airway plugin:install <module>
airway generate [action|api|model|migration|service|cmd] [params]
airway schema:dump
airway schema:show
airway upload /path/to/file
airway repl
airway version                                           # 或 -v / --version；打印 VERSION 文件内容
```

不带参数运行 `airway` 会打印用法说明。

## 创建新项目

```bash
airway new myapp                    # 目录名：myapp
airway new github.com/me/myapp      # module 路径；目录取路径最后一段
```

`airway new` 会以框架仓库的 `app/` 骨架为模板生成一个新项目，自动从
`.env.example` 生成 `.env`，执行 `go mod tidy`，并打印后续步骤：

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

与下面的命令不同，`plugin:new` 用全局安装的 `airway` 即可运行——它只是写文件，
不依赖编译期注册。

Plugin 是通过 `plugins.go` 中的 blank import 启用的可选功能模块（见
[Plugin 扩展机制](plugin.md)）：

```bash
go run . plugin:list           # 列出已注册的 Plugin 及挂载路径
go run . plugin:install <module> # 把 Plugin 内嵌的 SQL 迁移复制到 db/migrate
```

Plugin 在编译期注册，所以这些命令需要通过项目二进制运行（在项目目录中执行
`go run . ...`）：全局安装的 `airway` 只能列出/安装编译进它自身的 Plugin。

`plugin:install` 的参数是 Plugin 的模块路径（如 `github.com/daqing/airway-im-plugin`)，
插件名从路径最后一段推导（与 `plugin:new` 相同）。它会为复制的迁移文件分配新的时间戳，并跳过已安装的文件；复制后它们就是
普通迁移，由 `db:migrate` / `db:rollback` / `db:status` 统一管理。

## REPL

```bash
go run . repl
```

REPL 只能看到编译进当前二进制、通过 `github.com/daqing/airway/lib/replreg`
注册的模型——项目模型的 init 通过 `app/models` 的 `registerREPLModel` 注册
（该函数委托给 `lib/replreg`）。因此在项目中请使用 `go run . repl`；全局安装的
`airway repl` 只能看到框架自带的模型。

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
