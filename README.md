About
=====

Airway is a full-stack API framework written in Go, inspired by Ruby on Rails.

**[查看中文文档](https://github.com/daqing/airway/blob/main/docs/zh-CN/README.md)**

**[SQL Builder DSL Guide (Chinese)](https://github.com/daqing/airway/blob/main/docs/zh-CN/sql-builder.md)**

**[CLI Scaffolding Guide](https://github.com/daqing/airway/blob/main/docs/cli.md)**

Get Started
===========

## 1. Setup project skeleton

Clone the repo

```bash
$ git clone https://github.com/daqing/airway.git
```

## 2. Configure local development environment

### Create `.env` file

```bash
$ cp .env.example .env
```

This file defines a few environment variables:

**AIRWAY_DB_DSN**

The URL string for connecting to the application database.

Examples: `postgres://daqing@localhost:5432/airway`, `sqlite://./tmp/airway.db`, `mysql://root:passwd@127.0.0.1:3306/airway?charset=utf8mb4`

### Database Examples

Use the same application code and switch databases by changing only the DSN.

PostgreSQL:

```env
AIRWAY_DB_DSN="postgres://daqing:passwd@127.0.0.1:5432/airway"
```

SQLite 3:

```env
AIRWAY_DB_DSN="sqlite://./tmp/airway.db"
```

SQLite 3 in-memory:

```env
AIRWAY_DB_DSN="sqlite://:memory:"
```

MySQL 8:

```env
AIRWAY_DB_DSN="mysql://root:passwd@127.0.0.1:3306/airway?charset=utf8mb4"
```

MySQL also accepts the native Go driver DSN format:

```env
AIRWAY_DB_DSN="root:passwd@tcp(127.0.0.1:3306)/airway?charset=utf8mb4&parseTime=true"
```

Notes:

- Airway infers the database driver directly from the DSN.
- Basic CRUD flows are intended to work across PostgreSQL, SQLite 3, and MySQL 8 with the same Builder and repository APIs.
- Some advanced SQL Builder helpers are still PostgreSQL-oriented, especially ARRAY, JSONB, and a few lateral/window-heavy expressions.

**AIRWAY_PORT**

The port to listen on.

Example: `1900`

**TZ**

The timezone of the server

Example: `Asia/Shanghai`

## 3. Use the built-in CLI scaffolder

Airway now includes the former `awcli` functionality directly in the main command.
`airway cli ...` automatically attempts to load `.env` from the current project root during local development.

Quick examples:

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

Migration commands read `AIRWAY_DB_DSN` first and fall back to the legacy `AIRWAY_PG`.

See the full CLI guide at [docs/cli.md](/Users/daqing/mzevo/open-source/airway/docs/cli.md).

## 4. Start local development server

Run `just` from the project root directory to start the local
development server.

---

# Repository API Reference

A unified query layer with ActiveRecord-style association support.

- [Model Definition](#model-definition)
- [CRUD Operations](#crud-operations)
- [Preload (Eager Loading)](#preload-eager-loading)
- [Joins](#joins)
- [Rails Migration Guide](#rails-migration-guide)

## Model Definition

### Basic Model

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

### Model with Associations

```go
type User struct {
    ID        int64     `db:"id"`
    Name      string    `db:"name"`
    Email     string    `db:"email"`
    Profile   *Profile  // HasOne association
    Posts     []*Post   // HasMany association
}

func (User) TableName() string {
    return "users"
}

// Define associations
func (User) Relations() map[string]repo.Relation {
    return map[string]repo.Relation{
        "Profile": repo.HasOne(Profile{}, "UserID"),
        "Posts":   repo.HasMany(Post{}, "UserID"),
    }
}

type Profile struct {
    ID     int64  `db:"id"`
    UserID int64  `db:"user_id"`
    Bio    string `db:"bio"`
    User   *User  // BelongsTo association
}

func (Profile) TableName() string {
    return "profiles"
}

type Post struct {
    ID       int64      `db:"id"`
    UserID   int64      `db:"user_id"`
    Title    string     `db:"title"`
    Content  string     `db:"content"`
    Author   *User      // BelongsTo association
    Comments []*Comment // HasMany association
}

func (Post) TableName() string {
    return "posts"
}

func (Post) Relations() map[string]repo.Relation {
    return map[string]repo.Relation{
        "Author":   repo.NewBelongsTo(User{}, "UserID"),
        "Comments": repo.HasMany(Comment{}, "PostID"),
    }
}
```

## CRUD Operations

All CRUD operations use the database connection configured via `AIRWAY_DB_DSN`.

### Create

```go
// Create a single record
user, err := repo.CreateFrom[User](sql.H{
    "name":  "John Doe",
    "email": "john@example.com",
})
```

### Read

```go
// Find all records
users, err := repo.FindAll[User]()

// Find by conditions
users, err := repo.FindBy[User](sql.H{"active": true})

// Find single record
user, err := repo.FindOneBy[User](sql.H{"email": "john@example.com"})

// Find by ID
user, err := repo.FindByID[User](1)

// Check if exists
exists, err := repo.ExistsWhere[User](sql.H{"email": "john@example.com"})

// Count records
count, err := repo.CountWhere[User](sql.H{"active": true})

// Count all
total, err := repo.CountEvery[User]()
```

### Update

```go
// Update by ID
err := repo.UpdateByID[User](1, sql.H{"name": "Jane Doe"})

// Update with condition
err := repo.UpdateWhere[User](
    sql.H{"status": "inactive"},
    sql.Eq("last_login_at", nil),
)

// Update all records
err := repo.UpdateEvery[User](sql.H{"updated_at": "2024-01-01"})
```

### Delete

```go
// Delete by ID
err := repo.DeleteByID[User](1)

// Delete with condition
err := repo.DeleteWhere[User](sql.H{"status": "banned"})

// Delete all (use with caution!)
err := repo.DeleteEvery[User]()
```

## Preload (Eager Loading)

Preload solves the N+1 query problem by loading associations efficiently.

### Basic Preload

```go
// Without Preload (N+1 problem) - DON'T DO THIS
users, _ := repo.FindBy[User](sql.H{})
for _, user := range users {
    posts, _ := repo.FindBy[Post](sql.H{"user_id": user.ID}) // N queries!
    user.Posts = posts
}

// With Preload (only 2 queries)
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Posts").Exec(&users)
```

### Preload Multiple Associations

```go
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Profile", "Posts").Exec(&users)

// Access loaded data
for _, user := range users {
    _ = user.Profile.Bio    // Already loaded
    for _, post := range user.Posts {
        _ = post.Title      // Already loaded
    }
}
```

### Chained Preload

```go
// Load nested associations: Users -> Posts -> Comments
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Posts").
    ThenPreload("Profile").
    ThenPreload("Comments").
    Exec(&users)
```

### Conditional Preload

```go
// Only load approved comments
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Posts").
    ThenPreloadCond("Comments", sql.Eq("approved", true)).
    Exec(&users)

// Complex conditions
users, _ := repo.FindBy[User](sql.H{})
err := repo.PreloadCond("Posts", sql.And(
    sql.Eq("published", true),
    sql.Gte("created_at", "2024-01-01"),
)).Exec(&users)
```

## Joins

Use Joins when filtering based on associated table data.

### Join Types

```go
// Inner Join - only users with profiles
results, err := repo.Join(User{}).Joins("Profile").Find()

// Left Join - all users, with or without profiles
results, err := repo.Join(User{}).LeftJoins("Profile").Find()

// Right Join
results, err := repo.Join(User{}).RightJoins("Profile").Find()

// Full Join
results, err := repo.Join(User{}).FullJoins("Profile").Find()
```

### Join with Conditions

```go
// Join with conditions on the joined table
results, err := repo.Join(User{}).
    Joins("Posts", sql.Gt("posts.views", 100)).
    Find()

// Complex conditions
results, err := repo.Join(User{}).
    LeftJoins("Posts", sql.And(
        sql.Eq("posts.published", true),
        sql.Gte("posts.created_at", "2024-01-01"),
    )).
    Find()
```

### Join with Where, Order, Pagination

```go
results, err := repo.Join(User{}).
    Joins("Posts").
    Joins("Profile").
    Where(sql.Eq("users.active", true)).
    Where(sql.Gt("profiles.age", 18)).
    OrderBy("users.name ASC").
    Page(1, 20).
    Find()

// Alternative pagination
results, err := repo.Join(User{}).
    Joins("Posts").
    Limit(10).
    Offset(20).
    Find()
```

### Count with Joins

```go
count, err := repo.Join(User{}).
    Joins("Posts").
    Where(sql.Eq("posts.published", true)).
    Count()
```

### Scan Results into Structs

```go
var users []*User
err := repo.Join(User{}).
    LeftJoins("Profile").
    FindInto(&users)

// Access preloaded data
for _, user := range users {
    if user.Profile != nil {
        fmt.Println(user.Profile.Bio)
    }
}
```

## Rails Migration Guide

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

---

Repo REPL
=========

Airway now includes a built-in REPL for exercising `lib/repo` directly against a configured database.

Start it with the app database from `AIRWAY_DB_DSN`:

```bash
go run . repl
```

Or point it at a different database explicitly:

```bash
go run . repl --driver sqlite3 --dsn ./tmp/airway.db
go run . repl --dsn sqlite://./tmp/airway.db
```

Supported commands:

- `tables`
- `driver`
- `help`
- `exit`

Examples:

```text
repo.FindOne("users", cond.Eq("id", 1))
repo.Find("users", pg.Select("*").Where(cond.Eq("id", 1)))
repo.Find("users", cond.AllOf(cond.Eq("enabled", true), cond.Like("email", "%@example.com")))
repo.Insert("users", pg.H{"email": "dev@example.com", "enabled": true})
repo.Update("users", pg.H{"enabled": false}, cond.Eq("id", 1))
repo.Delete("users", cond.Eq("id", 1))
repo.Preview(pg.Select("*").From("users").Where(cond.Eq("id", 1)))
pg.Select("*").From("users").Where(cond.Eq("id", 1))
```

Available namespaces in the REPL are `repo`, `sql`, `pg`, `mysql`, and `sqlite`.

`repo.Find` / `repo.FindOne` / `repo.Count` / `repo.Exists` accept either a built statement, or `table + condition` arguments.

They also accept `table + stmt`; if the stmt has not bound a table yet, the REPL will bind that table before execution.

`repo.Update` and `repo.Delete` reject full-table writes unless the last argument is explicit `true`.

If you enter a builder expression directly, the REPL prints the compiled SQL and bind args.
