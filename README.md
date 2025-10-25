About
=====

Airway is a full-stack API framework written in Go, inspired by Ruby on Rails.

**[查看中文文档](https://github.com/daqing/airway/blob/main/docs/zh-CN/README.md)**

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

**AIRWAY_PG**

The URL string for connecting to PostgreSQL.

Example: `postgres://daqing@localhost:5432/airway`

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
