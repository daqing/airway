About
=====

Airway is a full-stack API framework written in Go, inspired by Ruby on Rails.

**[查看中文文档](https://github.com/daqing/airway/blob/main/docs/zh-CN/README.md)**

Get Started
===========

## 1. Setup project skeleton

Use `gonew` to create a new project based on `airway`:

```bash
$ gonew github.com/daqing/airway example.com/foo/bar
```

Replace `example.com/foo/bar` with your real module name.

## 2. Configure local development environment

### Create `.env` file

```bash
$ cp .env.example .env
```

This file defines a few environment variables:

**AIRWAY_PG_URL**

The URL string for connecting to PostgreSQL.

Example: `postgres://daqing@localhost:5432/airway`

**AIRWAY_PORT**

The port to listen on.

Example: `1900`

**AIRWAY_ROOT**

The full path to current project directory.

Example: `/Users/daqing/open-source/airway`

**TZ**

The timezone of the server

Example: `Asia/Shanghai`

## 3. Start local development server

Run `just` from the project root directory to start the local
development server.
