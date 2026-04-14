# Airway CLI 脚手架使用说明

Airway 已经内置脚手架命令，可以直接通过主命令调用：

```bash
go run . cli ...
```

执行 `airway cli ...` 时，Airway 会优先自动加载当前项目根目录下的 `.env` 文件。

如果你已经编译出二进制，也可以这样调用：

```bash
./airway cli ...
```

## 命令总览

```bash
airway cli db:create
airway cli db:drop
airway cli db:migrate [version]
airway cli db:rollback [step]
airway cli db:status
airway cli generate [action|api|model|migration|service|cmd] [params]
airway cli plugin install /path/to/project
airway cli schema:dump
airway cli schema:show
```

## 代码生成命令

### 生成 API 模块

```bash
go run . cli generate api admin
```

会创建：

- `app/api/admin_api/routes.go`
- `app/api/admin_api/index_action.go`

适合在你准备新增一个 API 命名空间时使用。

### 在已有 API 模块里生成 action

```bash
go run . cli generate action admin show
```

会创建：

- `app/api/admin_api/show_action.go`

适合在现有 API 目录下继续新增接口处理函数。

### 生成 model

```bash
go run . cli generate model post
```

会创建：

- `app/models/post.go`

生成内容默认包含：

- `ID`、`CreatedAt`、`UpdatedAt`
- `TableName()`
- 供 REPL 使用的 `registerREPLModel`

### 生成 service

```bash
go run . cli generate service post title:string published:bool
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
go run . cli generate cmd post title published
```

会创建：

- `cmd/post.go`

这个生成器适合给项目补充围绕 service 的命令行辅助函数。

### 生成迁移文件

```bash
go run . cli generate migration create_posts
```

会在以下目录生成新的 SQL migration：

- `db/migrate`

## 数据库迁移命令

### 执行全部待运行迁移

```bash
go run . cli db:migrate
```

### 迁移到指定版本

```bash
go run . cli db:migrate 20260327120000
```

### 回滚最近一次迁移

```bash
go run . cli db:rollback
```

### 按步数回滚

```bash
go run . cli db:rollback 3
```

### 查看迁移状态

```bash
go run . cli db:status
```

迁移相关命令读取数据库连接串的顺序为：

1. `AIRWAY_DB_DSN`
2. 兼容旧项目时回退到 `AIRWAY_PG`

在本地开发场景下，这些环境变量通常可以直接写在项目根目录的 `.env` 文件里，`airway cli ...` 会自动读取。
迁移命令会复用 Airway 当前 DSN 所对应的数据库类型，因此支持项目当前支持的 PostgreSQL、MySQL 和 SQLite。

## 安装插件到其他 Airway 项目

如果你当前仓库是一个插件项目，可以把它安装到另一个 Airway 项目：

```bash
go run . cli plugin install /path/to/project
```

该命令会复制：

- 当前项目的 `./app/*` 到目标项目的 `app/`
- 当前项目的 `./cmd/*` 到目标项目的 `cmd/`
- 当前项目的 `./db/migrate/*.sql` 到目标项目的 `db/migrate/`

复制 migration 文件时，会自动补一个新的时间戳前缀，避免版本号冲突。

## 实战示例

下面用一个最小例子，演示如何从零生成一个 `posts` 功能模块。

### 第 1 步：生成 migration

```bash
go run . cli generate migration create_posts
```

然后编辑 `db/migrate/` 下面新生成的 SQL 文件，写入表结构。

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
go run . cli db:migrate
```

### 第 2 步：生成 model

```bash
go run . cli generate model post
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
go run . cli generate service post title:string published:bool
```

这会创建 `app/services/post.go`，里面带有基础 CRUD 方法。

### 第 4 步：生成 API 模块

```bash
go run . cli generate api post
go run . cli generate action post create
go run . cli generate action post show
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
	"github.com/daqing/airway/app/api/up_api"
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
go run .
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
