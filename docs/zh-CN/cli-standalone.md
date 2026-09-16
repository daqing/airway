# 独立的 Airway 命令行工具

Airway 现在是一个可以直接通过 `go install` 安装的独立命令行工具。一个
`airway` 二进制文件覆盖项目的完整生命周期：脚手架新项目、生成代码、管理
数据库、启动服务器。

```bash
go install github.com/daqing/airway@latest
```

这会把 `airway` 命令安装到你的 Go bin 目录（请确保 `$(go env GOPATH)/bin`
已在 `PATH` 中）。在任何 Airway 项目内，同样的命令也可以用
`go run . <command>` 的方式运行。

## 快速上手：从零到运行中的应用

```bash
airway new myapp                    # 或：airway new github.com/me/myapp，或本地路径 /path/to/myapp
cd myapp
# 编辑 .env（已从 .env.example 自动生成）—— 配置 DSN 和 PORT
airway db:create
airway db:migrate
go run . server                     # 启动 HTTP 服务器
```

`airway new` 会基于内嵌模板生成完整的项目骨架（路由、模型、视图、
WebSocket、存储、Docker 配置），把模板中的 `{{module}}` 占位符替换为你的
module 路径，并自动执行 `go mod tidy`。目标目录取 module 路径的最后一段
（`github.com/me/myapp` → `./myapp`），且该目录不能是非空目录。

## 二进制文件如何分发命令

`airway` 二进制首先是一个 CLI：

- `airway server` 启动 HTTP 服务器（要求设置 `AIRWAY_ENV`；当
  `AIRWAY_ENV=local` 时会自动加载 `.env`）。
- 其他所有参数都会分发给 CLI。所有 CLI 命令会自动加载当前目录下的
  `.env`，因此请在项目根目录运行它们。
- 不带参数运行 `airway` 会打印用法说明。

框架仓库本身行为一致：`go run . server` 启动服务器，`go run . <command>`
运行 CLI 命令。（由 `airway new` 生成的项目，其二进制不带参数运行时也会
直接启动服务器。）

## 命令一览

```bash
airway new <module-path | directory>                    # 生成新项目骨架
airway server                                           # 启动 HTTP 服务器
airway generate [action|api|model|migration|service|cmd] [params]
airway db:create
airway db:drop
airway db:migrate [version]
airway db:rollback [step]
airway db:status
airway schema:dump
airway schema:show
airway plugin:new <module-path>                         # 生成新的 Plugin 模块骨架
airway plugin:list
airway plugin:install <module>
airway upload [key] /path/to/file
airway repl                                             # 仅项目二进制可用
airway version                                           # 或 -v / --version；打印 VERSION 文件内容
airway help                                             # 或 -h / --help
```

`generate` 可以缩写为 `g`（`airway g api admin`）。

### 代码生成器

生成器会读取当前目录 `go.mod` 中的 module 路径，因此生成的代码导入的是
*你自己*项目的包——请在项目根目录运行。生成器不会覆盖已存在的文件。

```bash
airway generate api admin                       # app/api/admin_api/（路由 + index action）
airway generate action admin show               # 在已有 API 模块中新增 action
airway generate model post                      # app/models/post.go（含 REPL 注册）
airway generate service post title:string       # app/services/ 下的 CRUD service
airway generate cmd post title published        # cmd/ 下的自定义 CLI 辅助命令
airway generate migration create_posts          # db/migrate/ 下的 SQL up/down 文件对
```

执行 `generate api` 之后，记得把生成的 `Routes(...)` 注册到
`config/routes.go` 中。

### 数据库与迁移

迁移文件是 `db/migrate/` 下的纯 SQL 文件对：

- `<时间戳>_<名称>.up.sql` —— 前向迁移
- `<时间戳>_<名称>.down.sql` —— 回滚迁移

因为迁移是纯 SQL 文件，独立 CLI 无需编译项目本身即可迁移*任何* Airway
项目。每个迁移在事务中执行，已应用的版本记录在 `schema_migrations` 表中，
`db:migrate` / `db:rollback` 会自动刷新 `db/schema.json` 快照。迁移支持
PostgreSQL、MySQL 和 SQLite，数据库驱动由 DSN 的 scheme 自动推断。

```bash
airway db:migrate                # 应用所有未执行的迁移
airway db:migrate 20260327120000 # 迁移到指定版本
airway db:rollback               # 回滚最近一个迁移
airway db:rollback 3             # 回滚三步
airway db:status                 # 查看每个迁移的 applied/pending 状态
```

数据库命令按以下顺序读取第一个已设置的环境变量作为 DSN：`DSN`、
`AIRWAY_DSN`、`AIRWAY_DB_DSN`（旧版）、`AIRWAY_PG`（旧版）。由于 CLI 会自动
加载 `.env`，在其中配置 `DSN` 即可。

旧的 Go DSL 迁移机制（`schema.RegisterChange`）仍然受支持，但 DSL 迁移只有
编译进执行它的二进制时才会生效。如果 CLI 在 `db/migrate` 下发现时间戳命名
的 `.go` 迁移文件，会打印警告——建议改用 SQL 迁移，以便独立 CLI 执行。

### Schema 快照

```bash
airway schema:dump   # 检查数据库并写入 db/schema.json
airway schema:show   # 打印当前的 db/schema.json
```

### Plugin（插件）

生成一个新的 Plugin 模块骨架（用全局安装的 `airway` 即可运行——它只是写文件）：

```bash
airway plugin:new im                              # 目录：im，Plugin 名称：im
airway plugin:new github.com/me/airway-im-plugin  # 名称从路径最后一段推导
```

Plugin 是通过 `plugins.go` 中的空白导入启用的可选功能模块
（参见 [plugin.md](plugin.md)）：

```bash
go run . plugin:list           # 列出已注册的 Plugin 及挂载路径
go run . plugin:install <module> # 把 Plugin 内嵌的 SQL 迁移复制到 db/migrate/
```

Plugin 在编译期注册，因此请通过**项目二进制**运行这些命令
（`go run . ...`）：全局安装的 `airway` 只能看到编译进它自身的 Plugin。
`plugin:install` 会为复制的迁移分配新的时间戳，并跳过已安装的文件；之后
它们就是普通迁移，由 `db:migrate` / `db:rollback` / `db:status` 管理。

### 文件上传

```bash
airway upload /tmp/foo.png                # 存储 key 由路径推导（tmp/foo.png）
airway upload images/foo.png /tmp/foo.png # 显式指定存储 key
```

上传走 `.env` 中配置的存储（`STORAGE_DRIVER`、`STORAGE_ROOT`、云端凭证等）。

### REPL

```bash
go run . repl
```

REPL 只能看到编译进当前二进制的模型——项目模型通过 `app/models` 中的
`registerREPLModel` 注册（底层委托给 `lib/replreg`）。请在你的项目内使用
`go run . repl`；全局安装的 `airway repl` 只能看到框架内置的模型。

## 全局 `airway` 与项目二进制的区别

有些命令依赖编译进当前二进制的代码（模型、Plugin、Go DSL 迁移），请为每条
命令选择正确的二进制：

| 命令 | 全局 `airway` | 项目二进制（`go run . ...`） |
| --- | --- | --- |
| `new`、`generate`、`server` | 可用 | 可用 |
| `plugin:new` | 可用 | 可用 |
| `db:create` / `db:drop` / `db:migrate` / `db:rollback` / `db:status` | 可用（SQL 迁移） | 可用 |
| `schema:dump` / `schema:show`、`upload`、`version` | 可用 | 可用 |
| `repl` | 仅框架内置模型 | **推荐**——能看到你的项目模型 |
| `plugin:list` / `plugin:install` | 仅编译进 `airway` 自身的 Plugin | **推荐**——能看到你项目的 Plugin |

## 兼容性

0.5 之前的 `airway cli <command>` 形式仍可作为 `airway <command>` 的别名
使用。

另请参阅：[../cli.md](../cli.md) 获取完整的脚手架指南和逐步特性示例。
