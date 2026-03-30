# 关于 Airway

Airway 是使用 go 语言开发的全栈 Web 框架，受到 Ruby on Rails 框架的启发。

相关文档：

- [SQL Builder DSL 使用文档](./sql-builder.md)
- [CLI 脚手架使用说明](./cli.md)

在开发 `Airway` 之前，作者有 10 年以上 Rails 开发经验，对于 Rails 框架的优点和缺点，有比较深入的理解。

Rails 框架的优点之一，在于清晰固定的目录结构。这样的好处是，对于接触一个新的 Rails 项目，可以很快的上手。

Rails 框架的优点其二，在于【约定胜于配置】的设计思想，其默认的规则，往往就满足了开发需求，而不需要写大量的配置文件。

Rails 框架的缺点，有以下几点：

1. 缺少业务逻辑层抽象，导致业务逻辑散落在控制器和 Model 之间。对于复杂的项目，你也许能看到几百行的 Model 定义。
2. 不够模块化。
3. 有些设计，属于黑魔法，对于第一次接触的人来说，有点难以掌握。

另外，由于 Rails 是基于 Ruby 语言开发的，其命令行工具，在生成框架代码时，运行速度非常慢（相比于 go 来说）。

所以，在吸取了 Rails 框架的优点，结合 Go 语言的特性，针对性的改进 Rails 框架的缺点，就这样，我开发出了 **`Airway`**。

# 快速上手

提示：目前 `airway` 还处于开发阶段，以下内容仅适用于当前的版本。如果有不完善的地方，或者好的点子和改进意见，欢迎提 issue。

## 1. 克隆项目代码

```bash
$ git clone https://github.com/daqing/airway.git
```

## 2. 搭建开发环境

#### 2.1

首先，项目使用 [just](https://github.com/casey/just) 来执行一些脚本命令。

关于如何安装 `just` 请参考其中文文档：[https://github.com/casey/just/blob/master/README.中文.md](https://github.com/casey/just/blob/master/README.中文.md)

除了 just, 项目还用到了以下软件：

- air: [github.com/cosmtrek/air](https://github.com/cosmtrek/air) 热重载 Go 应用的工具

- overmind: [github.com/DarthSim/overmind](https://github.com/DarthSim/overmind) Process manager for Procfile-based applications and tmux

对于 macOS 系统，只需要执行:

```bash
$ just install-deps
```

就可以把上述依赖的软件安装好。


#### 2.2

其次，项目使用了 `.env` 作为配置。

需要创建 `.env` 文件：

```bash
$ cp .env.example .env
```

这个文件，定义了几个环境变量，说明如下：

* AIRWAY_DB_DSN

  * 应用数据库连接字符串，支持 PostgreSQL、SQLite 3 和 MySQL 8.4。

  * PostgreSQL 示例: `postgres://daqing:passwd@127.0.0.1:5432/airway`

  * MySQL 示例: `mysql://daqing:passwd@127.0.0.1:3306/airway?charset=utf8mb4`

  * SQLite 示例: `sqlite://./tmp/airway.db`

#### 2.2.1 常见数据库配置示例

在当前版本中，应用层代码不需要改动，只需要切换 DSN 配置，就可以在 PostgreSQL、SQLite 3 和 MySQL 8 之间切换。

PostgreSQL:

```env
AIRWAY_DB_DSN="postgres://daqing:passwd@127.0.0.1:5432/airway"
```

SQLite 3 文件数据库:

```env
AIRWAY_DB_DSN="sqlite://./tmp/airway.db"
```

SQLite 3 内存数据库:

```env
AIRWAY_DB_DSN="sqlite://:memory:"
```

MySQL 8:

```env
AIRWAY_DB_DSN="mysql://root:passwd@127.0.0.1:3306/airway?charset=utf8mb4"
```

MySQL 也支持 Go 驱动原生 DSN:

```env
AIRWAY_DB_DSN="root:passwd@tcp(127.0.0.1:3306)/airway?charset=utf8mb4&parseTime=true"
```

补充说明：

- Airway 会根据 DSN 自动识别数据库驱动。
- 当前通用 CRUD 能力已经覆盖 PostgreSQL、SQLite 3 和 MySQL 8。
- `lib/sql` 中仍然有一部分明显偏 PostgreSQL 的高级 DSL，例如 ARRAY、JSONB、部分 `LATERAL`/复杂表达式写法；这些场景暂时不承诺三库完全一致。

* AIRWAY_PORT

  * 服务器监听的端口，默认为 `"1900"`

* TZ
  * 当前服务器的时区
  * 默认值为: `Asia/Shanghai`

#### 2.3

使用 Airway 内置脚手架命令

现在原来的 `awcli` 功能已经直接集成到主命令里，不需要再单独安装额外 CLI。
本地开发时，`airway cli ...` 会自动尝试加载当前项目根目录下的 `.env`。

常见示例：

```bash
go run . cli generate api admin
go run . cli generate action admin show
go run . cli generate model post
go run . cli generate service post title:string published:bool
go run . cli generate cmd post title published
go run . cli generate migration create_posts
go run . cli migrate
go run . cli rollback 1
go run . cli status
go run . cli plugin install /path/to/project
```

迁移相关命令优先读取 `AIRWAY_DB_DSN`，同时兼容旧的 `AIRWAY_PG`。

完整说明请查看 [CLI 脚手架使用说明](./cli.md)。

#### 2.4

修改 `Dockerfile`，把里面的`airway`，替换为项目名称。

```
FROM alpine

WORKDIR /app

RUN mkdir /app/bin

COPY ./bin/airway /app/bin

ENV AIRWAY_ENV=production
ENV AIRWAY_PORT=1900
ENV TZ="Asia/Shanghai"

EXPOSE 1900

CMD ["/app/bin/airway"]
```

假设你的项目目录是 `foo_site`，那么，当执行 go build 时，所生成的二进制名称就是 `foo_site`。

那么，你需要把上面内容中的 `airway`，替换为 `foo_site`


## 3. 启动本地开发服务器

执行以下命令：

```
$ just
```

就可以启动本地开发服务器。

根据你的 `.env` 中配置的端口，就可以访问对应的网址。

假设你配置的端口是 **2023**, 那么，访问 [http://localhost:2023](http://localhost:2023) 即可。

---

# 数据库操作 API 参考

Airway 提供了类似 Rails ActiveRecord 风格的 API，支持关联查询和 N+1 查询优化。

- [模型定义](#模型定义)
- [CRUD 操作](#crud-操作)
- [预加载 Preload](#预加载-preload)
- [关联查询 Joins](#关联查询-joins)
- [Rails 迁移对照表](#rails-迁移对照表)

## 模型定义

### 基础模型

```go
type User struct {
    ID        int64   `db:"id"`
    Name      string  `db:"name"`
    Email     string  `db:"email"`
    CreatedAt string  `db:"created_at"`
}

func (User) TableName() string {
    return "users"
}
```

### 带关联的模型

```go
type User struct {
    ID        int64     `db:"id"`
    Name      string    `db:"name"`
    Email     string    `db:"email"`
    Profile   *Profile  // HasOne 关联
    Posts     []*Post   // HasMany 关联
}

func (User) TableName() string {
    return "users"
}

// 定义关联
func (User) Relations() map[string]repo.Relation {
    return map[string]repo.Relation{
        "Profile": repo.NewHasOne(Profile{}, "UserID"),
        "Posts":   repo.NewHasMany(Post{}, "UserID"),
    }
}

type Profile struct {
    ID     int64  `db:"id"`
    UserID int64  `db:"user_id"`  // 外键
    Bio    string `db:"bio"`
    User   *User  // BelongsTo 关联
}

func (Profile) TableName() string {
    return "profiles"
}

type Post struct {
    ID       int64      `db:"id"`
    UserID   int64      `db:"user_id"`  // 外键
    Title    string     `db:"title"`
    Content  string     `db:"content"`
    Author   *User      // BelongsTo 关联
    Comments []*Comment // HasMany 关联
}

func (Post) TableName() string {
    return "posts"
}

func (Post) Relations() map[string]repo.Relation {
    return map[string]repo.Relation{
        "Author":   repo.NewBelongsTo(User{}, "UserID"),
        "Comments": repo.NewHasMany(Comment{}, "PostID"),
    }
}
```

## CRUD 操作

所有 CRUD 操作都通过 `AIRWAY_DB_DSN` 配置的连接执行。

### 创建

```go
// 创建单条记录
user, err := repo.CreateFrom[User](sql.H{
    "name":  "John Doe",
    "email": "john@example.com",
})
```

### 查询

```go
// 查询所有记录
users, err := repo.FindAll[User]()

// 根据条件查询
users, err := repo.FindBy[User](sql.H{"active": true})

// 查询单条记录
user, err := repo.FindOneBy[User](sql.H{"email": "john@example.com"})

// 根据 ID 查询
user, err := repo.FindByID[User](1)

// 检查是否存在
exists, err := repo.ExistsWhere[User](sql.H{"email": "john@example.com"})

// 统计记录数
count, err := repo.CountWhere[User](sql.H{"active": true})

// 统计所有记录
total, err := repo.CountEvery[User]()
```

### 更新

```go
// 根据 ID 更新
err := repo.UpdateByID[User](1, sql.H{"name": "Jane Doe"})

// 根据条件更新
err := repo.UpdateWhere[User](
    sql.H{"status": "inactive"},
    sql.Eq("last_login_at", nil),
)

// 更新所有记录
err := repo.UpdateEvery[User](sql.H{"updated_at": "2024-01-01"})
```

### 删除

```go
// 根据 ID 删除
err := repo.DeleteByID[User](1)

// 根据条件删除
err := repo.DeleteWhere[User](sql.H{"status": "banned"})

// 删除所有记录（谨慎使用！）
err := repo.DeleteEvery[User]()
```

## 预加载 Preload

预加载解决 N+1 查询问题，只需 2 次查询即可加载所有关联数据。

### 基础预加载

```go
// 不使用预加载（N+1 问题）- 不要这样做
users, _ := repo.FindBy[User](sql.H{})
for _, user := range users {
    posts, _ := repo.FindBy[Post](sql.H{"user_id": user.ID}) // N 次查询！
    user.Posts = posts
}

// 使用预加载（只需 2 次查询）
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Posts").Exec(&users)
```

### 预加载多个关联

```go
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Profile", "Posts").Exec(&users)

// 访问已加载的数据
for _, user := range users {
    _ = user.Profile.Bio    // 已加载
    for _, post := range user.Posts {
        _ = post.Title      // 已加载
    }
}
```

### 链式预加载

```go
// 加载多层关联：Users -> Posts -> Comments
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Posts").
    ThenPreload("Profile").
    ThenPreload("Comments").
    Exec(&users)
```

### 条件预加载

```go
// 只加载已批准的评论
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Posts").
    ThenPreloadCond("Comments", sql.Eq("approved", true)).
    Exec(&users)

// 复杂条件
users, _ := repo.FindBy[User](sql.H{})
err := repo.PreloadCond("Posts", sql.And(
    sql.Eq("published", true),
    sql.Gte("created_at", "2024-01-01"),
)).Exec(&users)
```

## 关联查询 Joins

使用 Joins 根据关联表数据进行过滤。

### Join 类型

```go
// Inner Join - 只返回有 Profile 的用户
results, err := repo.Join(User{}).Joins("Profile").Find()

// Left Join - 返回所有用户，包括没有 Profile 的
results, err := repo.Join(User{}).LeftJoins("Profile").Find()

// Right Join
results, err := repo.Join(User{}).RightJoins("Profile").Find()

// Full Join
results, err := repo.Join(User{}).FullJoins("Profile").Find()
```

### 带条件的 Join

```go
// 对关联表添加条件
results, err := repo.Join(User{}).
    Joins("Posts", sql.Gt("posts.views", 100)).
    Find()

// 复杂条件
results, err := repo.Join(User{}).
    LeftJoins("Posts", sql.And(
        sql.Eq("posts.published", true),
        sql.Gte("posts.created_at", "2024-01-01"),
    )).
    Find()
```

### Join + Where + 排序 + 分页

```go
results, err := repo.Join(User{}).
    Joins("Posts").
    Joins("Profile").
    Where(sql.Eq("users.active", true)).
    Where(sql.Gt("profiles.age", 18)).
    OrderBy("users.name ASC").
    Page(1, 20).
    Find()

// 另一种分页方式
results, err := repo.Join(User{}).
    Joins("Posts").
    Limit(10).
    Offset(20).
    Find()
```

### Join 统计

```go
count, err := repo.Join(User{}).
    Joins("Posts").
    Where(sql.Eq("posts.published", true)).
    Count()
```

### 扫描到结构体

```go
var users []*User
err := repo.Join(User{}).
    LeftJoins("Profile").
    FindInto(&users)

// 访问已加载的数据
for _, user := range users {
    if user.Profile != nil {
        fmt.Println(user.Profile.Bio)
    }
}
```

## Rails 迁移对照表

| Rails ActiveRecord | Airway Repo |
|-------------------|-------------|
| `User.find(id)` | `repo.FindByID[User](id)` |
| `User.find_by(email: e)` | `repo.FindOneBy[User](sql.H{"email": e})` |
| `User.where(active: true)` | `repo.FindBy[User](sql.H{"active": true})` |
| `User.all` | `repo.FindAll[User]()` |
| `User.create(attrs)` | `repo.CreateFrom[User](attrs)` |
| `User.update(id, attrs)` | `repo.UpdateByID[User](id, attrs)` |
| `User.delete(id)` | `repo.DeleteByID[User](id)` |
| `User.joins(:profile)` | `repo.Join(User{}).Joins("Profile")` |
| `User.left_joins(:profile)` | `repo.Join(User{}).LeftJoins("Profile")` |
| `User.includes(:posts)` | `repo.Preload("Posts").Exec(&users)` |
| `User.includes(:profile, :posts)` | `repo.Preload("Profile", "Posts").Exec(&users)` |
| `User.includes(posts: :comments)` | `repo.Preload("Posts").ThenPreload("Comments").Exec(&users)` |
| `User.count` | `repo.CountEvery[User]()` |
| `User.where(active: true).count` | `repo.CountWhere[User](sql.H{"active": true})` |
| `User.exists?(id)` | `repo.ExistsWhere[User](sql.H{"id": id})` |
