# SQL Builder DSL 使用文档

本文档介绍 Airway 当前内置的 SQL Builder 体系，重点说明如何在项目中构建 SQL 查询，而不依赖重量级 ORM。

当前实现已经从“一个 Builder 尝试兼容所有数据库”的方向，调整为“一个核心 DSL + 三个方言 Builder”的结构：

- `lib/sql`：底层通用 DSL 与内部实现
- `lib/sql/pg`：PostgreSQL Builder
- `lib/sql/mysql`：MySQL Builder
- `lib/sql/sqlite`：SQLite Builder

这套 Builder 体系的目标是：

- 保持 SQL 语义清晰，可预测地生成 SQL
- 使用命名参数，避免手写占位符和参数顺序错误
- 让业务代码在编译期就明确绑定数据库方言
- 把数据库能力边界直接体现在 API 暴露面上
- 让 PostgreSQL、SQLite 3、MySQL 8 各自按真实能力使用，而不是在运行时碰撞出错
- 通过表/字段引用减少裸字符串 SQL 片段
- 在简单场景保持足够轻量，在复杂场景提供更强约束


## 0. 先说结论

现在推荐的使用原则非常明确：

- 业务代码不要再默认直接依赖 `lib/sql`
- 业务代码应该根据项目选定的数据库，明确导入一个方言包：`pg`、`mysql` 或 `sqlite`
- `lib/sql` 现在主要承担底层实现、共用类型和共用 helper 的职责
- `lib/repo` 已经改为接收 `sql.Stmt` 接口，因此三套 Builder 都可以直接传给 `repo.Find`、`repo.Insert`、`repo.Update`、`repo.Delete`

这样做的直接收益是：

- 如果某个数据库不支持某个能力，那么对应 Builder 就不暴露这个方法
- 代码会在编译期失败，而不是运行时才发现 SQL 不可执行
- API 的暴露面本身就成为数据库能力文档

例如：

- `pg.Builder` 支持 `DistinctOn`、`JoinLateral`、`ForShare`、`UpdateFrom`、`Using`、`ILike`、JSONB/ARRAY helper
- `mysql.Builder` 不支持这些方法，因此不会误用
- `sqlite.Builder` 支持 `Returning`，但不支持 `LATERAL JOIN`、`FOR UPDATE`、`ILIKE`

### 0.1 你应该导入哪个包

PostgreSQL 项目：

```go
import pg "github.com/daqing/airway/lib/sql/pg"
```

MySQL 项目：

```go
import mysql "github.com/daqing/airway/lib/sql/mysql"
```

SQLite 项目：

```go
import sqlite "github.com/daqing/airway/lib/sql/sqlite"
```

查询条件（通用，不绑定任何方言）：

```go
import "github.com/daqing/airway/lib/sql/cond"
```

只有在以下情况，才应该直接依赖 `lib/sql`：

- 实现底层 SQL DSL 本身
- 编写方言 Builder 的包装层
- 编写 `repo` 层这类需要接收统一接口的基础设施代码

### 0.2 当前结构

核心结构如下：

- [lib/sql/base.go](/Users/daqing/mzevo/open-source/airway/lib/sql/base.go)：定义 `Stmt`、`H`、基础类型
- [lib/sql/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder.go)：底层 Builder 实现
- [lib/sql/pg/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/pg/builder.go)：PostgreSQL 方言包装
- [lib/sql/mysql/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/mysql/builder.go)：MySQL 方言包装
- [lib/sql/sqlite/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/sqlite/builder.go)：SQLite 方言包装
- [lib/repo/query.go](/Users/daqing/mzevo/open-source/airway/lib/repo/query.go)：执行前的数据库方言转换和命名参数编译

### 0.3 能力矩阵

下表按“是否暴露为 Builder API”来理解，而不是数据库理论上是否完全支持某种 SQL 变体。

| 能力 | pg | mysql | sqlite |
| --- | --- | --- | --- |
| 基础 SELECT / INSERT / UPDATE / DELETE | `Yes` | `Yes` | `Yes` |
| `RETURNING` | `Yes` | `No` | `Yes` |
| `DISTINCT ON` | `Yes` | `No` | `No` |
| `JOIN LATERAL` | `Yes` | `No` | `No` |
| `FULL JOIN` | `Yes` | `No` | `Yes` |
| `FOR UPDATE` | `Yes` | `Yes` | `No` |
| `FOR SHARE` | `Yes` | `No` | `No` |
| `UPDATE ... FROM` | `Yes` | `No` | `No` |
| `DELETE ... USING` | `Yes` | `No` | `No` |
| `ON CONFLICT (columns)` | `Yes` | `Yes` | `Yes` |
| `ON CONFLICT ON CONSTRAINT` | `Yes` | `No` | `No` |
| `ILIKE / NOT ILIKE` | `Yes` | `No` | `No` |
| JSONB helper | `Yes` | `No` | `No` |
| ARRAY helper | `Yes` | `No` | `No` |
| `INTERSECT ALL / EXCEPT ALL` | `Yes` | `No` | `Yes` |

补充说明：

- `mysql.OnConflictDoNothing(...)` 最终会在 `repo` 层转换为 `INSERT IGNORE`
- `mysql.OnConflictDoUpdate(...)` 最终会在 `repo` 层转换为 `ON DUPLICATE KEY UPDATE`
- `sqlite.Returning(...)` 依赖 SQLite 3.35+
- `sqlite.FullJoin(...)` 依赖 SQLite 3.39+

### 0.4 为什么不再追求一个统一 SQL 层

因为不同数据库之间的真实差异并不只是语法皮肤差异，而是能力边界差异：

- PostgreSQL 有 `JSONB`、`ARRAY`、`DISTINCT ON`、`LATERAL JOIN`
- MySQL 没有 `RETURNING`，冲突更新语法也不是 `ON CONFLICT`
- SQLite 虽然支持 `RETURNING`，但不支持 `FOR UPDATE` / `FOR SHARE`

如果继续把所有能力都堆进同一个对外 Builder，调用端会面临两个问题：

- IDE 自动补全会暴露大量当前数据库根本不能使用的方法
- 开发者只能靠文档或运行时报错记住哪些能力不能用

现在的结构是把“数据库能力差异”前移到 API 层，直接通过包和方法可见性表达出来。

### 0.5 推荐开发习惯

更合理的原则是：

- 单表、简单 CRUD、字段很少的查询：继续使用字符串 API 完全可以接受
- 多表 join、别名较多、schema-qualified table、复杂排序/返回列：优先使用表/字段引用写法
- PostgreSQL 独有能力：只在 `pg` 包中书写和维护
- 如果项目未来可能切数据库，不要在通用查询中混入 PostgreSQL 专有 helper

当前 DSL 仍然建议采用两层风格：

- 默认写法：简单、直接
- 表/字段写法：在复杂 SQL 中提供更高可读性和安全性

为了降低样板代码，同时避免 `Ref` 和 `Col` 这类缩写过于抽象，现在更推荐使用更直白的命名：

- `TableOf("users")`
- `TableAlias("users", "u")`
- `users.Field("id")`
- `users.AllFields()`

例如：

```go
u := sql.TableAlias("users", "u")

b := sql.SelectFields(u.AllFields(), u.Field("email")).
    FromTable(u).
    Where(sql.FieldEq(u.Field("enabled"), true)).
    OrderBy(u.Field("id").Desc())
```

如果你希望 API 尽量大白话，推荐统一使用：

- `users.Field("id")`
- `users.AllFields()`
- `SelectFields(...)`
- `FieldEq(...)`

这些命名在三个方言包里都做了重新导出，因此你可以这样写：

```go
users := pg.TableAlias("users", "u")

b := pg.SelectFields(users.AllFields()).
    FromTable(users).
    Where(cond.FieldEq(users.Field("enabled"), true))
```


兼容性说明：

- `Ref(...)`、`RefAs(...)` 仍然保留
- `Col(...)`、`users.Col(...)` 仍然保留
- `SelectRefs(...)`、`EqRef(...)`、`GroupByRefs(...)`、`ReturningRefs(...)` 这类旧名字也仍然保留
- 但后续文档示例优先使用 `TableOf(...)`、`TableAlias(...)`、`Field(...)`、`AllFields()`、`SelectFields(...)`、`FieldEq(...)`

如果你在 Go 代码里查看类型签名，现在对外更推荐看到的是：

- `TableName`
- `FieldName`

它们分别是表引用和字段引用的对外类型名；原来的 `TableRef`、`ColumnRef` 继续保留兼容。


## 1. 核心概念

从实现上看，现在 Builder 分为 5 层：

- 方言层：`pg`、`mysql`、`sqlite`
- 语句层：`Select`、`Insert`、`Update`、`Delete`
- 条件层：`Where`、`AllOf`、`AnyOf`、`Compare`、`FieldEq` 等
- 表达式层：`Func`、`Array`、`Any`、`JSONGetText`、`Raw` 等
- 引用层：`TableOf`、`TableAlias`、`Field`、`TableFor`、`FieldFor`

业务代码最常见的入口应该是这三个文件：

- [lib/sql/pg/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/pg/builder.go)
- [lib/sql/mysql/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/mysql/builder.go)
- [lib/sql/sqlite/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/sqlite/builder.go)

底层实现和共用类型仍然在这些文件中：

- [lib/sql/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder.go)
- [lib/sql/op.go](/Users/daqing/mzevo/open-source/airway/lib/sql/op.go)
- [lib/sql/expr.go](/Users/daqing/mzevo/open-source/airway/lib/sql/expr.go)
- [lib/sql/identifiers.go](/Users/daqing/mzevo/open-source/airway/lib/sql/identifiers.go)
- [lib/sql/table.go](/Users/daqing/mzevo/open-source/airway/lib/sql/table.go)


## 2. 基本用法

### 2.1 生成 SQL 和参数

所有 Builder 最终都通过 `ToSQL()` 导出 SQL 和命名参数：

```go
b := pg.Select("*").From("users").Where(cond.Eq("enabled", true))

query, args := b.ToSQL()
```

返回值：

- `query`: 最终 SQL 字符串
- `args`: `map[string]any`，命名参数集合

如果你通过 `repo` 层执行，则通常不需要自己手工消费这两个返回值。

现在仓储层统一接收的是 `sql.Stmt` 接口，而不是 `*sql.Builder`。也就是说：

- `*pg.Builder` 可以直接传进去
- `*mysql.Builder` 可以直接传进去
- `*sqlite.Builder` 可以直接传进去
- 原始 `*sql.Builder` 也仍然可以传进去

仓储层中已经封装好了这一步，参考：

- [lib/repo/find.go](/Users/daqing/mzevo/open-source/airway/lib/repo/find.go)
- [lib/repo/insert.go](/Users/daqing/mzevo/open-source/airway/lib/repo/insert.go)
- [lib/repo/update.go](/Users/daqing/mzevo/open-source/airway/lib/repo/update.go)
- [lib/repo/delete.go](/Users/daqing/mzevo/open-source/airway/lib/repo/delete.go)

例如：

```go
user, err := repo.FindOne[User](repo.CurrentDB(), pg.Select("*").From("users").Where(cond.Eq("id", 1)))
```


### 2.2 基本 SELECT

```go
b := sql.Select("*").
    From("users").
    Where(sql.Eq("enabled", true)).
    OrderBy("id DESC").
    Limit(20).
    Offset(40)
```

更推荐在复杂查询中使用表/字段写法：

```go
users := sql.TableAlias("users", "u")

b := sql.SelectFields(users.AllFields()).
    FromTable(users).
    Where(sql.FieldEq(users.Field("enabled"), true)).
    OrderBy(users.Field("id").Desc()).
    Limit(20).
    Offset(40)
```


### 2.3 基本 INSERT

```go
b := sql.Insert(sql.H{
    "title":     "hello",
    "completed": false,
}).Into("todos")
```

表写法：

```go
todos := sql.TableOf("todos")

b := sql.Insert(sql.H{
    "title":     "hello",
    "completed": false,
}).IntoTable(todos)
```


### 2.4 基本 UPDATE

```go
b := sql.Update("users").
    Set(sql.H{"name": "new name"}).
    Where(sql.Eq("id", 1))
```

表/字段写法：

```go
users := sql.TableOf("users")

b := sql.UpdateTable(users).
    Set(sql.H{"name": "new name"}).
    Where(sql.FieldEq(users.Field("id"), 1))
```


### 2.5 基本 DELETE

```go
b := sql.Delete().
    From("users").
    Where(sql.Eq("id", 1))
```

表/字段写法：

```go
users := sql.TableOf("users")

b := sql.DeleteFrom(users).
    Where(sql.FieldEq(users.Field("id"), 1))
```

说明：

- 从这一节开始，后面的示例为了简洁，仍然大量使用 `sql.xxx` 作为抽象写法
- 在真实业务代码中，请把 `sql` 替换为你项目选定的方言包，例如 `pg.xxx`、`mysql.xxx`、`sqlite.xxx`
- 如果示例里出现了 PostgreSQL 独有 helper，例如 `ILike`、`Array`、`JSONGetText`，那么它只应该出现在 `pg` 包代码里


## 3. 表/字段引用

这是复杂查询时更推荐的调用方式，但不是强制要求。

### 3.1 表引用

在 Go 类型层面，这类值通常会显示为 `TableName`。

```go
users := sql.TableOf("users")
usersWithSchema := sql.TableOf("public", "users")
aliasedUsers := sql.TableAlias("users", "u")
schemaAliasedUsers := sql.TableOf("public", "users").As("u")
```

表引用会在生成 SQL 时自动做 identifier quoting：

```sql
"public"."users" AS "u"
```


### 3.2 字段引用

在 Go 类型层面，这类值通常会显示为 `FieldName`。

```go
users := sql.TableAlias("users", "u")

idCol := users.Field("id")
emailCol := users.Field("email")
allCols := users.AllFields()
aliased := users.Field("email").As("user_email")
```

辅助方法：

- `Asc()`
- `Desc()`
- `WithoutAlias()`：去掉 alias，只保留原引用

例如：

```go
users.Field("created_at").Desc()
```


### 3.3 Builder 中使用表/字段引用

```go
users := sql.TableAlias("users", "u")
posts := sql.TableAlias("posts", "p")

b := sql.SelectFields(
    users.Field("id"),
    users.Field("email"),
    posts.Field("title").As("post_title"),
).
    FromTable(users).
    LeftJoinTable(posts, sql.Compare(posts.Field("user_id"), "=", users.Field("id"))).
    GroupByFields(users.Field("id"), users.Field("email"), posts.Field("title"))
```


## 4. 条件 DSL

### 4.1 旧风格条件

适合兼容已有代码：

- `Eq`
- `NotEq`
- `Gt`
- `Gte`
- `Lt`
- `Lte`
- `Like`
- `NotLike`
- `ILike`（仅 `pg`）
- `NotILike`（仅 `pg`）
- `In`
- `NotIn`
- `Between`
- `NotBetween`
- `IsNull`
- `IsNotNull`

示例：

```go
sql.AllOf(
    sql.Eq("status", "open"),
    sql.Gte("priority", 10),
    sql.Not(sql.IsNull("deleted_at")),
)
```


### 4.2 表字段条件

这是现在更推荐的方式：

- `FieldEq`
- `FieldNotEq`
- `FieldGt`
- `FieldGte`
- `FieldLt`
- `FieldLte`
- `FieldLike`
- `FieldILike`（仅 `pg`）
- `Compare`

示例：

```go
users := sql.TableAlias("users", "u")

cond := sql.AllOf(
    sql.FieldEq(users.Field("status"), "active"),
    sql.FieldGte(users.Field("score"), 60),
    sql.Compare(users.Field("role"), "=", sql.Any(sql.Array("admin", "editor"))),
)
```


### 4.3 布尔组合

- `AllOf(...)`
- `AnyOf(...)`
- `Not(...)`

示例：

```go
cond := sql.AllOf(
    sql.FieldEq(users.Field("enabled"), true),
    sql.AnyOf(
        sql.FieldEq(users.Field("role"), "admin"),
        sql.FieldEq(users.Field("role"), "editor"),
    ),
)
```


### 4.4 按表构建 map 条件

如果你的代码习惯传 `map[string]any`，可以直接使用：

- `MatchFields(table, vals)`
- `MatchTable(table, vals)`

示例：

```go
users := sql.TableAlias("users", "u")

cond := sql.MatchFields(users, sql.H{
    "email":  "dev@example.com",
    "status": "active",
})
```

这会把 map 中的 key 自动绑定到 `users` 表作用域下的列引用，而不是继续用裸字符串列名。

如果你的业务代码已经明确绑定某个数据库，推荐始终从方言包调用这些 helper，例如：

```go
cond := cond.MatchFields(users, pg.H{
    "email":  "dev@example.com",
    "status": "active",
})
```


## 5. 表达式 DSL

常用表达式 helper：

- `Raw(sql)`
- `Expr(sql)`
- `ExprNamed(sql, args)`
- `Column(name)`
- `Func(name, args...)`
- `Op(left, operator, right)`
- `Cast(value, type)`
- `Excluded(column)`
- `Default()`
- `SubQuery(query)`

其中：

- `Array(values...)`、`Any(value)`、`AllExpr(value)` 只在 `pg` 包中暴露
- `Excluded(column)` 在三个方言包中都可见，但最常见于 `OnConflictDoUpdate(...)`
- `SubQuery(query)` 接收的是当前方言 Builder，而不是其他方言 Builder

示例：

```go
sql.Compare(
    sql.Func("COUNT", posts.Field("id")),
    ">",
    0,
)
```

```go
sql.Compare(
    users.Field("role"),
    "=",
    sql.Any(sql.Array("admin", "editor")),
)
```


## 6. SELECT DSL

这一节列出的很多能力都来自底层 core builder，但不同方言包只会选择性暴露其中一部分。

### 6.1 选择列

- `Select("*")`
- `SelectColumns(...)`
- `SelectFields(...)`
- `Columns(...)`
- `Fields(...)`

示例：

```go
b := sql.SelectFields(users.Field("id"), users.Field("email")).
    Columns(`COUNT("p"."id") AS post_count`)
```


### 6.2 FROM / JOIN

- `From(tableName)`
- `FromTable(table)`
- `FromExpr(expr)`
- `FromSubQuery(query, alias)`
- `Join(...)`
- `JoinTable(...)`
- `LeftJoin(...)`
- `LeftJoinTable(...)`
- `RightJoin(...)`
- `RightJoinTable(...)`
- `FullJoin(...)`
- `FullJoinTable(...)`
- `CrossJoin(...)`
- `CrossJoinTable(...)`
- `JoinExpr(...)`
- `JoinLateral(...)`
- `LeftJoinLateral(...)`

可用性：

- `FullJoin(...)` / `FullJoinTable(...)`：仅 `pg` 和 `sqlite`
- `JoinLateral(...)` / `LeftJoinLateral(...)`：仅 `pg`

示例：

```go
users := sql.TableAlias("users", "u")
posts := sql.TableAlias("posts", "p")

b := sql.SelectFields(users.Field("id"), posts.Field("title")).
    FromTable(users).
    LeftJoinTable(posts, sql.FieldEq(posts.Field("user_id"), users.Field("id")))
```


### 6.3 DISTINCT / GROUP / HAVING / WINDOW

- `Distinct()`
- `DistinctOn(fields...)`
- `GroupBy(fields...)`
- `GroupByFields(columns...)`
- `Having(cond)`
- `Window(definitions...)`

可用性：

- `DistinctOn(fields...)`：仅 `pg`
- `Window(definitions...)`：`pg`、`mysql`、`sqlite` 都提供

示例：

```go
b := sql.SelectFields(users.Field("id")).
    Columns(`COUNT("p"."id") AS post_count`).
    FromTable(users).
    LeftJoinTable(posts, sql.FieldEq(posts.Field("user_id"), users.Field("id"))).
    GroupByFields(users.Field("id")).
    Having(sql.Compare(sql.Func("COUNT", posts.Field("id")), ">", 0)).
    Window("w AS (PARTITION BY id)")
```


### 6.4 排序和分页

- `OrderBy(...)`
- `OrderBys(...)`
- `Limit(...)`
- `Offset(...)`
- `Page(page, perPage)`

表/字段写法推荐：

```go
b.OrderBy(users.Field("id").Desc())
```


### 6.5 FOR UPDATE / FOR SHARE

- `For(clause)`
- `ForUpdate()`
- `ForShare()`

可用性：

- `For(clause)`：仅 `pg`
- `ForUpdate()`：`pg`、`mysql`
- `ForShare()`：仅 `pg`


### 6.6 CTE

- `With(name, query)`
- `WithRecursive(name, query)`

示例：

```go
activeUsers := sql.SelectFields(sql.Field("user_id")).
    FromTable(sql.TableOf("sessions")).
    Where(sql.FieldEq(sql.Field("revoked"), false))

b := sql.SelectFields(users.Field("id")).
    With("active_users", activeUsers).
    FromTable(users)
```


### 6.7 集合查询

- `Union(query)`
- `UnionAll(query)`
- `Intersect(query)`
- `IntersectAll(query)`
- `Except(query)`
- `ExceptAll(query)`

可用性：

- `Union`、`UnionAll`、`Intersect`、`Except`：三个方言包都提供
- `IntersectAll`、`ExceptAll`：仅 `pg`、`sqlite`

示例：

```go
active := sql.SelectFields(sql.Field("id")).FromTable(sql.TableOf("users")).Where(sql.FieldEq(sql.Field("enabled"), true))
invited := sql.SelectFields(sql.Field("user_id").As("id")).FromTable(sql.TableOf("invitations")).Where(sql.FieldEq(sql.Field("accepted"), true))

b := active.UnionAll(invited).OrderBy(sql.Field("id").Desc())
```


## 7. INSERT DSL

### 7.1 单行插入

```go
b := sql.Insert(sql.H{
    "title": "demo",
}).IntoTable(sql.TableOf("todos"))
```


### 7.2 多行插入

```go
b := sql.InsertRows(
    sql.H{"id": 1, "name": "alpha"},
    sql.H{"id": 2, "name": "beta"},
).Into("users")
```


### 7.3 INSERT ... SELECT

- `InsertColumns(...)`
- `FromSelect(query, columns...)`

示例：

```go
source := sql.SelectFields(sql.Field("id"), sql.Field("email")).
    FromTable(sql.TableOf("users")).
    Where(sql.FieldEq(sql.Field("confirmed"), true))

b := sql.Insert(nil).
    IntoTable(sql.TableOf("newsletter_subscribers")).
    FromSelect(source, "user_id", "email")
```


### 7.4 ON CONFLICT

- `OnConflictDoNothing(columns...)`
- `OnConflictOnConstraintDoNothing(constraint)`
- `OnConflictDoUpdate(columns, vals)`
- `OnConflictOnConstraintDoUpdate(constraint, vals)`

可用性：

- `OnConflictDoNothing(columns...)`：三个方言包都提供
- `OnConflictDoUpdate(columns, vals)`：三个方言包都提供
- `OnConflictOnConstraintDoNothing(...)`：仅 `pg`
- `OnConflictOnConstraintDoUpdate(...)`：仅 `pg`

MySQL 特别说明：

- `mysql.OnConflictDoNothing(...)` 在执行前会被转换成 `INSERT IGNORE`
- `mysql.OnConflictDoUpdate(...)` 在执行前会被转换成 `ON DUPLICATE KEY UPDATE`
- 这部分转换逻辑位于 [lib/repo/query.go](/Users/daqing/mzevo/open-source/airway/lib/repo/query.go)

示例：

```go
b := sql.InsertRows(
    sql.H{"id": 1, "name": "alpha"},
).Into("users").
    OnConflictDoUpdate([]string{"id"}, sql.H{
        "name":       sql.Excluded("name"),
        "updated_at": sql.Raw("NOW()"),
    })
```


### 7.5 RETURNING

- `Returning(fields...)`
- `ReturningFields(columns...)`
- `ReturningAll()`

可用性：

- `pg`：支持
- `sqlite`：支持
- `mysql`：不暴露


## 8. UPDATE DSL

- `Set(vals)`
- `UpdateFrom(tables...)`
- `Returning(...)`
- `ReturningFields(...)`

可用性：

- `Set(vals)`：三个方言包都提供
- `UpdateFrom(tables...)`：仅 `pg`
- `Returning(...)` / `ReturningFields(...)`：`pg`、`sqlite`

示例：

```go
users := sql.TableOf("users")

b := sql.UpdateTable(users).
    Set(sql.H{
        "name":       "new name",
        "updated_at": sql.Raw("NOW()"),
    }).
    Where(sql.FieldEq(users.Field("id"), 1)).
    ReturningFields(users.Field("id"), users.Field("name"))
```


## 9. DELETE DSL

- `DeleteFrom(table)`
- `DeleteKey(column)`
- `DeleteKeyField(field)`
- `Using(tables...)`
- `Returning(...)`
- `ReturningFields(...)`

可用性：

- `DeleteFrom(table)`、`DeleteKey(column)`、`DeleteKeyField(field)`：三个方言包都提供
- `Using(tables...)`：仅 `pg`
- `Returning(...)` / `ReturningFields(...)`：`pg`、`sqlite`

当 `DELETE` 语句带有 `ORDER BY / LIMIT / OFFSET` 时，Builder 会自动退化为：

```sql
DELETE FROM table
WHERE id IN (
  SELECT id FROM table
  ...
)
```

示例：

```go
events := sql.TableOf("audit", "events")

b := sql.DeleteFrom(events).
    DeleteKeyField(sql.Field("audit", "events", "event_id")).
    Where(sql.Compare(sql.Field("audit", "events", "kind"), "=", "login")).
    OrderBy(sql.Field("audit", "events", "created_at").Desc()).
    Limit(5).
    ReturningFields(sql.Field("audit", "events", "event_id"))
```


## 10. PostgreSQL JSONB / ARRAY helper

这一章是 PostgreSQL 专属能力，只应该在 `pg` 包中使用。

### 10.1 ARRAY

- `Array(...)`
- `Any(...)`
- `AllExpr(...)`
- `ArrayContains(left, right)` 对应 `@>`
- `ArrayContainedBy(left, right)` 对应 `<@`
- `ArrayOverlap(left, right)` 对应 `&&`

示例：

```go
sql.ArrayOverlap(users.Field("roles"), sql.Array("admin", "editor"))
```


### 10.2 JSONB

- `JSONGet(left, key)` 对应 `->`
- `JSONGetText(left, key)` 对应 `->>`
- `JSONPath(left, path...)` 对应 `#>`
- `JSONPathText(left, path...)` 对应 `#>>`
- `JSONContains(left, right)` 对应 `@>`
- `JSONContainedBy(left, right)` 对应 `<@`
- `JSONHasKey(left, key)` 对应 `?`
- `JSONHasAnyKeys(left, keys...)` 对应 `?|`
- `JSONHasAllKeys(left, keys...)` 对应 `?&`

示例：

```go
cond := sql.AllOf(
    sql.JSONHasKey(users.Field("profile"), "status"),
    sql.Compare(sql.JSONGetText(users.Field("profile"), "status"), "=", "active"),
    sql.JSONHasAnyKeys(users.Field("settings"), "beta", "dark_mode"),
)
```


## 11. 迁移指引

如果你的代码之前统一使用：

```go
import sql "github.com/daqing/airway/lib/sql"
```

现在建议按下面方式迁移。

### 11.1 业务代码迁移

PostgreSQL 项目：

```go
import pg "github.com/daqing/airway/lib/sql/pg"
```

把：

```go
sql.Select("*")
sql.Insert(...)
sql.Update(...)
sql.Delete()
sql.FieldEq(...)
```

替换为：

```go
pg.Select("*")
pg.Insert(...)
pg.Update(...)
pg.Delete()
cond.FieldEq(...)
```

MySQL 和 SQLite 项目同理，只是前缀替换为 `mysql` 或 `sqlite`。

### 11.2 `repo` 层调用无需特殊改造

之前：

```go
b := sql.Select("*").From("users")
user, err := repo.FindOne[User](repo.CurrentDB(), b)
```

现在：

```go
b := pg.Select("*").From("users")
user, err := repo.FindOne[User](repo.CurrentDB(), b)
```

原因是 `repo` 现在接收的是 `sql.Stmt` 接口，而不是固定的 `*sql.Builder`。

### 11.3 什么时候还可以继续用 `lib/sql`

以下场景继续直接使用 `lib/sql` 是合理的：

- 你在维护底层 Builder 本身
- 你在编写一个新的方言包装层
- 你在写基础设施代码，希望接收 `sql.Stmt` 而不关心具体数据库

普通业务查询不建议继续直接依赖 `lib/sql`，否则你会重新失去“按数据库能力裁剪 API”的收益。

### 11.4 常见迁移策略

建议这样逐步迁移：

1. 先按项目数据库类型，把 import 从 `lib/sql` 替换成对应方言包
2. 确认项目还能通过编译
3. 编译报错的位置，通常就是你正在调用该数据库不支持的能力
4. 再针对这些位置决定：改写 SQL，还是保留为 PostgreSQL 专有查询

这也是这次改造最重要的目标之一：让“不兼容能力”在编译阶段暴露出来。

## 12. 与项目封装层配合使用

项目已经把部分 helper 和 CRUD 封装迁到了表/字段 DSL：

- [lib/sql/table.go](/Users/daqing/mzevo/open-source/airway/lib/sql/table.go)
- [lib/db/find.go](/Users/daqing/mzevo/open-source/airway/lib/db/find.go)
- [lib/db/update.go](/Users/daqing/mzevo/open-source/airway/lib/db/update.go)
- [lib/db/delete.go](/Users/daqing/mzevo/open-source/airway/lib/db/delete.go)

例如：

```go
func FindById[T sql.Table](id sql.IdType) (*T, error) {
    var t T
    b := sql.FindByCond(t, sql.FieldEq(sql.FieldFor(t, "id"), id))
    return repo.FindOne[T](repo.CurrentDB(), b)
}
```

如果你在业务代码里继续封装自己的 repository/service，建议优先使用：

- `TableFor`
- `FieldFor`
- `FieldEq`
- `MatchTable`
- `SelectFields`
- `FromTable`

如果你的封装层不想绑定具体数据库，也可以像 `lib/repo` 一样，直接接收 `sql.Stmt` 作为参数类型。


## 13. 推荐实践

### 13.1 优先使用表/字段引用写法

复杂查询推荐：

```go
users := sql.TableAlias("users", "u")
sql.FieldEq(users.Field("id"), 1)
```

简单查询继续这样写也完全可以：

```go
sql.Eq("id", 1)
```

只有在别名、多表、schema、复杂 SQL 表达式开始增多时，再切换到这种表/字段引用写法，收益才最明显。


### 13.2 原始 SQL 只留给复杂表达式

适合用原始字符串的场景：

- 聚合别名
- 窗口函数
- 特别复杂的数据库函数表达式

例如：

```go
Columns(`ROW_NUMBER() OVER (ORDER BY "p"."created_at" DESC) AS rn`)
```


### 13.3 单表 map 条件优先用 `MatchTable`

如果你是从 HTTP 参数或 service 层拿到一个 `map[string]any`，不要直接继续拼字段名，优先使用：

```go
sql.MatchTable(t, vals)
```


### 13.4 用测试验证生成 SQL

这套 DSL 的一个重要优势，就是生成 SQL 足够稳定，因此很适合直接测试 query string 和 args。

可以参考：

- [lib/sql/builder_test.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder_test.go)


## 14. 当前限制

当前 DSL 已经支持大部分常见 PostgreSQL 查询，但仍然有一些能力还没有专门封装成更高层 API，例如：

- 聚合 `FILTER (...)`
- `NULLS FIRST / NULLS LAST`
- `ON CONFLICT ... WHERE`
- 更完整的窗口函数 DSL
- 更完整的 aggregate / alias builder

这些能力目前仍然可以通过 `Raw(...)`、`Columns(...)`、`Expr(...)`、`Compare(...)` 组合实现。


## 15. 参考示例

完整的调用示例可以直接看：

- [lib/sql/builder_test.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder_test.go)
- [lib/repo/crud_integration_test.go](/Users/daqing/mzevo/open-source/airway/lib/repo/crud_integration_test.go)
