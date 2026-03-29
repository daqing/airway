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
repo.FindOne("users", pg.Eq("id", 1))
repo.Find("users", pg.Select("*").Where(pg.Eq("id", 1)))
repo.Find("users", pg.AllOf(pg.Eq("enabled", true), pg.Like("email", "%@example.com")))
repo.Insert("users", pg.H{"email": "dev@example.com", "enabled": true})
repo.Update("users", pg.H{"enabled": false}, pg.Eq("id", 1))
repo.Delete("users", pg.Eq("id", 1))
repo.Preview(pg.Select("*").From("users").Where(pg.Eq("id", 1)))
pg.Select("*").From("users").Where(pg.Eq("id", 1))
```

Available namespaces in the REPL are `repo`, `sql`, `pg`, `mysql`, and `sqlite`.

`repo.Find` / `repo.FindOne` / `repo.Count` / `repo.Exists` accept either a built statement, or `table + condition` arguments.

They also accept `table + stmt`; if the stmt has not bound a table yet, the REPL will bind that table before execution.

`repo.Update` and `repo.Delete` reject full-table writes unless the last argument is explicit `true`.

If you enter a builder expression directly, the REPL prints the compiled SQL and bind args.
