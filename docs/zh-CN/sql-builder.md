# SQL Builder DSL 使用文档

本文档介绍 Airway 当前内置的 SQL Builder DSL，重点说明如何在项目中构建 PostgreSQL 查询，而不依赖重量级 ORM。

这套 DSL 的目标是：

- 保持 SQL 语义清晰，可预测地生成 SQL
- 使用命名参数，避免手写占位符和参数顺序错误
- 支持 PostgreSQL 常见查询能力
- 通过表/字段引用减少裸字符串 SQL 片段
- 在简单场景保持足够轻量，在复杂场景提供更强约束


## 0. 先说结论

这套表/字段引用能力值得保留，但不应该强制在所有查询里使用。

更合理的原则是：

- 单表、简单 CRUD、字段很少的查询：继续使用字符串 API 完全可以接受
- 多表 join、别名较多、schema-qualified table、复杂排序/返回列、JSONB/ARRAY 表达式：优先使用表/字段引用写法

也就是说，表引用这层能力应该是“复杂查询的稳定工具”，而不是“所有查询都必须写得很重”。

当前 DSL 建议采用两层风格：

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

SQL Builder 主要分为 4 层：

- 语句层：`Select`、`Insert`、`Update`、`Delete`
- 条件层：`Where`、`AllOf`、`AnyOf`、`Compare`、`FieldEq` 等
- 表达式层：`Func`、`Array`、`Any`、`JSONGetText`、`Raw` 等
- 引用层：`TableOf`、`TableAlias`、`Field`、`TableFor`、`FieldFor`

最常见的入口都在这些文件中：

- [lib/sql/builder.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder.go)
- [lib/sql/op.go](/Users/daqing/mzevo/open-source/airway/lib/sql/op.go)
- [lib/sql/expr.go](/Users/daqing/mzevo/open-source/airway/lib/sql/expr.go)
- [lib/sql/identifiers.go](/Users/daqing/mzevo/open-source/airway/lib/sql/identifiers.go)
- [lib/sql/table.go](/Users/daqing/mzevo/open-source/airway/lib/sql/table.go)


## 2. 基本用法

### 2.1 生成 SQL 和参数

所有 Builder 最终都通过 `ToSQL()` 导出 SQL 和命名参数：

```go
b := sql.Select("*").From("users").Where(sql.Eq("enabled", true))

query, args := b.ToSQL()
```

返回值：

- `query`: 最终 SQL 字符串
- `args`: `map[string]any`，用于传给 `pgx.NamedArgs`

仓储层中已经封装好了这一步，参考：

- [lib/repo/pg/find.go](/Users/daqing/mzevo/open-source/airway/lib/repo/pg/find.go)
- [lib/repo/pg/insert.go](/Users/daqing/mzevo/open-source/airway/lib/repo/pg/insert.go)
- [lib/repo/pg/update.go](/Users/daqing/mzevo/open-source/airway/lib/repo/pg/update.go)
- [lib/repo/pg/delete.go](/Users/daqing/mzevo/open-source/airway/lib/repo/pg/delete.go)


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
- `ILike`
- `NotILike`
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
- `FieldILike`
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


## 5. 表达式 DSL

常用表达式 helper：

- `Raw(sql)`
- `Expr(sql)`
- `ExprNamed(sql, args)`
- `Column(name)`
- `Func(name, args...)`
- `Op(left, operator, right)`
- `Cast(value, type)`
- `Array(values...)`
- `Any(value)`
- `AllExpr(value)`
- `Excluded(column)`
- `Default()`
- `SubQuery(query)`

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


## 8. UPDATE DSL

- `Set(vals)`
- `UpdateFrom(tables...)`
- `Returning(...)`
- `ReturningFields(...)`

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


## 11. 与项目封装层配合使用

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
    return pg.FindOne[T](pg.CurrentDB(), b)
}
```

如果你在业务代码里继续封装自己的 repository/service，建议优先使用：

- `TableFor`
- `FieldFor`
- `FieldEq`
- `MatchTable`
- `SelectFields`
- `FromTable`


## 12. 推荐实践

### 12.1 优先使用表/字段引用写法

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


### 12.2 原始 SQL 只留给复杂表达式

适合用原始字符串的场景：

- 聚合别名
- 窗口函数
- 特别复杂的数据库函数表达式

例如：

```go
Columns(`ROW_NUMBER() OVER (ORDER BY "p"."created_at" DESC) AS rn`)
```


### 12.3 单表 map 条件优先用 `MatchTable`

如果你是从 HTTP 参数或 service 层拿到一个 `map[string]any`，不要直接继续拼字段名，优先使用：

```go
sql.MatchTable(t, vals)
```


### 12.4 用测试验证生成 SQL

这套 DSL 的一个重要优势，就是生成 SQL 足够稳定，因此很适合直接测试 query string 和 args。

可以参考：

- [lib/sql/builder_test.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder_test.go)


## 13. 当前限制

当前 DSL 已经支持大部分常见 PostgreSQL 查询，但仍然有一些能力还没有专门封装成更高层 API，例如：

- 聚合 `FILTER (...)`
- `NULLS FIRST / NULLS LAST`
- `ON CONFLICT ... WHERE`
- 更完整的窗口函数 DSL
- 更完整的 aggregate / alias builder

这些能力目前仍然可以通过 `Raw(...)`、`Columns(...)`、`Expr(...)`、`Compare(...)` 组合实现。


## 14. 参考示例

完整的调用示例可以直接看：

- [lib/sql/builder_test.go](/Users/daqing/mzevo/open-source/airway/lib/sql/builder_test.go)
- [lib/repo/pg/db_test.go](/Users/daqing/mzevo/open-source/airway/lib/repo/pg/db_test.go)
