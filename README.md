About
=====

Airway is a full-stack API framework written in Go, inspired by Ruby on Rails.

**[查看中文文档](https://github.com/daqing/airway/blob/main/docs/zh-CN/README.md)**

**[SQL Builder DSL Guide (Chinese)](https://github.com/daqing/airway/blob/main/docs/zh-CN/sql-builder.md)**

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

## 3. Install `awcli`: the CLI for airway

```bash
go install github.com/daqing/airway-cli/cmd/awcli@latest
```

## 4. Start local development server

Run `just` from the project root directory to start the local
development server.
