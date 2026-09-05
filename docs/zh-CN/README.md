# Airway

一个受 Ruby on Rails 启发的全栈 API 框架，使用 Go 编写。它可以用**同一套应用代码**运行在 **PostgreSQL**、**MySQL 8** 和 **SQLite** 上——数据库驱动会在运行时根据 DSN 自动推断。

- **[English README](../README.md)**
- **[CLI 脚手架指南](cli.md)** / **[文件存储指南](storage.md)**
- **[SQL Builder DSL 指南](sql-builder.md)**

## Airway 是什么

Airway 既是**框架/库**，也是一个**可直接运行的应用程序骨架**：

- `lib/` 下可复用的分层——SQL Builder、Repository/ORM、迁移、存储、渲染、校验。
- 一个基于 Gin 的 HTTP 服务（`main.go` + `app/` + `config/`），内置脚手架 CLI、WebSocket 与 REPL，可直接 `go run .` 启动。

## 特性

- **基于泛型的 Repository**（`lib/repo`）：类型安全的 `FindBy[User]`、`CreateFrom[User]`、预加载（eager loading）、联表查询、事务。
- **方言感知的 SQL Builder**（`lib/sql` + `pg` / `mysql` / `sqlite` 方言包）；条件写法如 `sql.Eq`、`sql.AllOf`、`sql.Gt`。
- **基于 schema 的迁移**：通过 CLI 生成与应用迁移；SQLite 的表结构变更通过重建表处理。
- **统一的文件存储**（`lib/storage`）：本地目录、Amazon S3、Cloudflare R2 或腾讯云 COS——完全由配置决定，并提供 HTTP 上传/下载 API。
- **Gin Web 服务 + WebSocket** 发布/订阅。
- **脚手架 CLI**（`airway cli generate ...`、`db:migrate` 等）。
- **Repo REPL**：支持类型化扫描与 Go 表达式求值。
- **可选子路径前缀**（`URL_PREFIX`）：便于在反向代理后部署，例如 `http://host:1900/airway/...`。

## 快速开始

### 1. 获取代码并配置 `.env`

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
go run .   # 需要设置 AIRWAY_ENV（例如 AIRWAY_ENV=local）以及配置好的 DSN
```

或者使用热重载启动本地开发服务：

```bash
just dev
```

应用监听 `http://127.0.0.1:1900`（`GET /` 返回 `Hello, Airway!`，`GET /health` 返回 `UP`）。

## 配置

所有配置都通过环境变量完成（参见 `.env.example`）。每个值既可以使用短名，也可以使用其 `AIRWAY_` 别名，两者同时存在时以别名优先。

| 变量 | 说明 |
| --- | --- |
| `DSN` / `AIRWAY_DSN` | 数据库 URL。驱动由 scheme 推断（见下文）。 |
| `PORT` / `AIRWAY_PORT` | HTTP 监听端口（默认 `1900`）。 |
| `REDIS` / `AIRWAY_REDIS` | 可选的 Redis URL（缓存/队列）。 |
| `URL_PREFIX` / `AIRWAY_URL_PREFIX` | 可选的对外子路径前缀，例如 `/airway`。为空则在根路径提供服务。 |
| `AIRWAY_ENV` | `local` 会加载 `.env` 并使用 Gin 调试模式；其它值则运行 release 模式。 |
| `STORAGE_DRIVER` | `local`（默认）、`s3`、`r2` 或 `cos`。 |
| `STORAGE_ROOT` | 本地存储根目录（默认 `./data/storage`）。 |
| `STORAGE_*` | 云存储设置：`STORAGE_REGION`、`STORAGE_ENDPOINT`、`STORAGE_BUCKET`、`STORAGE_ACCESS_KEY`、`STORAGE_SECRET_KEY`、`STORAGE_PUBLIC_URL`（可选 CDN 基地址）。 |
| `TZ` | 服务器时区，例如 `Asia/Shanghai`。 |

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

- 基础的 CRUD 流程可跨 PostgreSQL、MySQL 8 与 SQLite 移植。
- 部分高级 SQL Builder 辅助函数（ARRAY、JSONB、一些 lateral/window 表达式）仍然偏向 PostgreSQL。
- SQLite 支持使用纯 Go 的 `modernc.org/sqlite` 驱动。

## HTTP 端点

注册于 `config/routes.go`：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/` | 首页（纯文本）。 |
| GET | `/health` | 健康检查。 |
| GET | `/ws` | WebSocket 连接。 |
| POST | `/ws/publish` | 向已连接的客户端发布消息（表单字段 `message`）。 |
| POST | `/api/v1/storage` | 上传文件（multipart `file`，可选 `dir`）。 |
| GET | `/api/v1/storage/*key` | 下载文件。 |
| DELETE | `/api/v1/storage/*key` | 删除文件。 |

当设置了 `URL_PREFIX` 时，以上所有端点也会在前缀路径下提供服务。例如设置 `URL_PREFIX="/airway"`：

```bash
curl http://127.0.0.1:1900/airway/health
curl -F "file=@report.pdf" http://127.0.0.1:1900/airway/api/v1/storage
```

客户端——包括 WebSocket 连接——都必须带上该前缀
（`ws://host:1900/airway/ws`）。本地存储接口返回的 URL 会自动加上前缀；云存储/CDN 的 URL 不受影响。

## CLI

所有命令都通过主程序运行；`cli` 命令会自动从项目根目录加载 `.env`：

```bash
airway cli generate api admin                  # 在 app/api/ 下新建 API namespace
airway cli generate action admin show          # 在已有 API 模块中新增 action
airway cli generate model post                 # 在 app/models/ 下新建 model
airway cli generate service post title:string  # 在 app/services/ 下新建 CRUD service
airway cli generate migration create_posts     # 在 db/migrate/ 下新建迁移
airway cli db:create | db:drop
airway cli db:migrate [version]                # 应用迁移
airway cli db:rollback [step]
airway cli db:status
airway cli schema:dump | schema:show           # 写入 / 读取 db/schema.json
airway cli upload [key] /path/to/file          # 通过已配置的存储上传文件
airway cli plugin install /path/to/project     # 把 app/、cmd/ 与迁移复制到另一个项目
```

数据库相关命令读取 `DSN`/`AIRWAY_DSN`；为了向后兼容，仍支持旧的 `AIRWAY_DB_DSN` 与 `AIRWAY_PG`。完整的[CLI 指南](cli.md)。

## Repository API（`lib/repo`）

### Model

Model 是带有 `db` 标签、并实现 `TableName()` 的结构体。关联关系通过 `Relations()` 声明。

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

以下辅助函数都使用启动时配置好的数据库（`repo.SetupDB`），并把 model 类型作为类型参数传入：

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

### Preload（预加载）

Preload 用少量查询代替 N+1 循环：

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

### Joins（联表查询）

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

直接针对已配置的数据库运行 `lib/repo`：

```bash
go run . repl                              # 使用已配置的 DSN
go run . repl --driver sqlite --dsn ./tmp/airway.db
```

命令：`help`、`driver`、`tables`、`exit`。直接输入一条 Go 表达式即可求值——Builder 会打印编译后的 SQL，`repo.*` 调用会真实执行数据库操作：

```text
repo.FindOne("users", sql.Eq("id", 1))
repo.Find[models.User](pg.Select("id").Where(sql.Eq("id", 1)))
repo.Insert[models.User](pg.H{"id": 1234, "email": "dev@example.com"})
repo.Update("users", pg.H{"enabled": false}, sql.Eq("id", 1))
repo.Delete("users", sql.Eq("id", 1))
pg.Select("*").From("users").Where(sql.Eq("id", 1))
```

可用的 namespace：`repo`、`sql`、`pg`、`mysql`、`sqlite`、`models`。
`repo.Find`/`FindOne`/`Count`/`Exists` 既可以接收构建好的语句，也可以接收 `表名 + 条件`；类型化调用支持匿名 struct 与应用 model。整表更新/删除必须以显式 `true` 作为最后一个参数。

## 文件存储

`lib/storage` 是基于本地目录或云后端（S3 / R2 / COS）的统一存储层。通过 `STORAGE_DRIVER` 选择后端：

```env
# 本地
STORAGE_DRIVER="local"
STORAGE_ROOT="./data/storage"

# 云存储（S3 兼容）：选择 s3 / r2 / cos
STORAGE_DRIVER="s3"
STORAGE_REGION="us-east-1"
STORAGE_BUCKET="my-bucket"
STORAGE_ACCESS_KEY="..."
STORAGE_SECRET_KEY="..."
# STORAGE_PUBLIC_URL="https://cdn.example.com"   # 可选 CDN 基地址
```

在代码中通过当前后端使用：

```go
store := storage.Current() // 启动时由 storage.Setup 注入

err := store.Put(ctx, "docs/report.pdf", storage.Object{
	Reader:      file,
	Size:        size,
	ContentType: "application/pdf",
})
rc, err := store.Get(ctx, "docs/report.pdf") // 使用后需 close
ok, err := store.Exists(ctx, "docs/report.pdf")
url, err := store.URL(ctx, "docs/report.pdf", 24*time.Hour)
err := store.Delete(ctx, "docs/report.pdf")
```

`URL()` 在本地后端返回应用自身的下载路径，在云后端返回 CDN 或预签名 URL。应用在同一存储层之上暴露了 REST API：

```bash
# 上传（multipart 字段 "file"，可选 "dir"）
curl -F "file=@report.pdf" -F "dir=docs" http://127.0.0.1:1900/api/v1/storage
# => {"key":"docs/202609/ab12cd....pdf","url":"/api/v1/storage/docs/202609/ab12cd....pdf","size":12345}

# 下载 / 删除
curl -O http://127.0.0.1:1900/api/v1/storage/docs/202609/ab12cd....pdf
curl -X DELETE http://127.0.0.1:1900/api/v1/storage/docs/202609/ab12cd....pdf
```

完整的[存储指南](storage.md)。

## 部署

该二进制是自包含的。使用以下命令构建纯 Go 镜像（无需 CGO 工具链）：

```bash
just docker        # 或：docker build -t airway .
docker run -p 1900:1900 -e AIRWAY_ENV=production -e DSN="sqlite:///app/tmp/airway.db" airway
```

部署前后运行迁移：

```bash
./airway cli db:migrate
```

Compose 示例见 [docs/docker-compose.yml.example](../docker-compose.yml.example)。如需在反向代理后面以子路径前缀提供服务，请设置 `URL_PREFIX`（例如 `/airway`）——参见上文[「HTTP 端点」](#http-端点)。

## 更多指南

- [CLI 脚手架指南](cli.md) · [文件存储指南](storage.md)
- [SQL Builder DSL 指南](sql-builder.md)
- [English README](../README.md)
