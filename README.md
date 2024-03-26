About
=====

Airway is a full-stack web framework written in Go, inspired by Ruby on Rails.

**[查看中文文档](https://github.com/daqing/airway/blob/main/docs/zh-CN/README.md)**

Get Started
===========

## 1. Setup project skeleton

Use `gonew` to create a new project based on `airway`:

```bash
$ gonew github.com/daqing/airway example.com/foo/bar
```

Replace `example.com/foo/bar` with your real module name.

## 2. Setup local development environment

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

**AIRWAY_PWD**

The full path to current project directory.

Example: `/Users/daqing/mindzee/open-source/airway`

### Prepare database

Run `just setup-db` to setup the database.

Run `just cli seed root root airway-root` to create the initial root account.

### Start local development server and login

Run `just` from the project root directory to start the local
development server.

Then you can login to the admin panel with:

```
username: root
password: airway-root
```
