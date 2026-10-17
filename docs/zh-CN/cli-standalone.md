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
# 编辑 .env（已从 .env.example 自动生成）—— 配置 DSN 和 LISTEN
airway db:create
airway db:migrate
go run . server                     # 启动 HTTP 服务器
```

`airway new` 会基于内嵌模板生成完整的项目骨架（路由、模型、视图、
WebSocket、存储、Docker 配置），把模板中的 `{{module}}` 占位符替换为你的
module 路径，并自动执行 `go mod tidy` 和 `airway js:install`（安装前端依赖）。
目标目录取 module 路径的最后一段
（`github.com/me/myapp` → `./myapp`），且该目录不能是非空目录。

## 二进制文件如何分发命令

`airway` 二进制首先是一个 CLI：

- `airway server` 启动 HTTP 服务器。必须设置 `AIRWAY_ENV`（进程环境或
  `.env` 均可）；`.env` 作为回退加载（进程环境优先），`AIRWAY_ENV=local`
  还会开启 Gin 调试模式。
- 其他所有参数都会分发给 CLI。所有 CLI 命令会自动加载当前目录下的
  `.env`，因此请在项目根目录运行它们。
- 不带参数运行 `airway` 会打印用法说明。

在项目内，全局安装的 `airway` 会识别宿主应用（`go.mod` 依赖
`github.com/daqing/airway`，且旁边有 `main.go`），并把所有项目级命令透明地
通过 `go run .` 重新执行——因此 Plugin、REPL 模型和 Go 代码迁移始终来自项目
自己的二进制。它会在 stderr 打印 `proxying to project binary` 提示。只有当全局
CLI 的版本与项目 `go.mod` 中 pin 的版本一致时才会代理；版本不一致会直接报错
终止，而不是静默使用项目二进制里较旧的 CLI 逻辑。`new`、`ssg:new`、
`theme:new`、`version`、`help` 和 `templates:compile` 始终在本地执行，在项目
之外运行的一切命令同样如此。

框架仓库本身行为一致：`go run . server` 启动服务器，`go run . <command>`
运行 CLI 命令。（由 `airway new` 生成的项目，其二进制不带参数运行时也会
直接启动服务器。）

## 命令一览

```bash
airway new [--local[=path]] <module-path | directory>     # 生成新项目骨架
airway server                                            # 启动 HTTP 服务器
airway generate [action|api|model|migration|service|island|scaffold|cmd] [params]
airway db:create
airway db:drop
airway db:migrate [version]
airway db:rollback [step]
airway db:status
airway schema:dump
airway schema:show
airway openapi:generate [--out path]                     # 写出 OpenAPI 3.2 文档
airway admin:generate [--force[=tables]] [config/admin.toml]
airway admin:root <username> <password>
airway admin:member <username> <password> [--role=editor|viewer]
airway desktop:init [--force]                            # 生成 Wails v3 桌面目标
airway ssg:new [--local[=path]] <module-path | directory> # 脚手架静态展示站点项目
airway ssg:build [--out dist]
airway ssg:serve [--addr 127.0.0.1:3000]
airway static:build [--out dist]                         # 导出 export.go 注册的应用页面
airway static:serve [--addr 127.0.0.1:3000]
airway theme:new [--local[=path]] <module-path | path>   # 脚手架站点主题模块
airway theme:install <module | /path>
airway templates:compile                                 # 重新生成 templ 视图
airway js:add <pkg>[@version]                            # 添加前端依赖（无需 Node）
airway js:install                                        # 按 js.pkg.json 安装到 vendor/
airway js:build                                          # 打包 app/assets/js 到 dist/
airway plugin:new <module-path | /path>                  # 生成新的 Plugin 模块骨架
airway plugin:list
airway plugin:install <module>
airway plugin:lint
airway upload [key] /path/to/file
airway repl                                              # 仅项目二进制可用
airway version                                           # 或 -v / --version；打印 VERSION 文件内容
airway help                                              # 或 -h / --help
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
airway generate island chart                    # app/assets/js/islands/ 下的交互式 island
airway generate scaffold post title:string      # 完整 CRUD：模型、迁移、API、页面、island
airway generate migration create_posts          # db/migrate/ 下的 Go DSL 迁移
```

执行 `generate api` 之后，记得把生成的 `Routes(...)` 注册到
`config/routes.go` 中。`generate scaffold` 会自动完成这一步，并立即编译新的
`.templ` 视图。

编辑 `app/views/` 下的 `.templ` 视图后，重新编译生成的 Go 代码（等价于
`go generate ./...`，它会执行 `generate.go` 里声明的 templ 指令）：

```bash
airway templates:compile       # 等价于 `go generate ./...`
```

### 数据库与迁移

`airway generate migration create_posts` 会在 `db/migrate/` 下写入一个带时间戳
的 Go DSL 迁移（例如 `<时间戳>_create_posts.go`，package `migrations`）。文件在
`init()` 中通过 `lib/migrate/schema` 注册自己，down 迁移由变更自动反向推导——
同一份文件可以在 PostgreSQL、MySQL 和 SQLite 上运行。DSL 无法表达的内容可以
退回原始 SQL：使用 `schema.Register(version, name, up, down)`，或在 change 内用
`m.Reversible(up, down)`。

Go DSL 迁移只有在被编译进执行它的二进制时才会运行：`db/migrate` 包必须被
`main.go` 空白导入（`airway new` 生成的项目已经包含该导入）。在项目内，全局
安装的 `airway` 会把迁移命令代理到 `go run .`，因此它们看到的迁移与服务器完全
一致。遗留的时间戳 SQL 文件对（`<version>_<name>.up.sql` / `.down.sql`）仍然会
被执行，旧项目可以继续工作。

每个迁移在事务中执行，已应用的版本记录在 `schema_migrations` 表中，
`db:migrate` / `db:rollback` 会自动刷新 `db/schema.json` 快照。迁移支持
PostgreSQL、MySQL 和 SQLite，数据库驱动由 DSN 的 scheme 自动推断。

```bash
airway db:migrate                # 应用所有未执行的迁移
airway db:migrate 20260327120000 # 迁移到指定版本
airway db:rollback               # 回滚最近一个迁移
airway db:rollback 3             # 回滚三步
airway db:status                 # 查看每个迁移的 applied/pending 状态
```

数据库命令按以下顺序读取第一个已设置的环境变量作为 DSN：`AIRWAY_DSN`、
`DSN`。由于 CLI 会自动加载 `.env`，在其中配置 `DSN` 即可。

### Schema 快照

```bash
airway schema:dump   # 检查数据库并写入 db/schema.json
airway schema:show   # 打印当前的 db/schema.json
```

### 前端依赖与打包

```bash
airway js:add preact                        # 添加依赖（最新版，精确锁定）
airway js:add @tanstack/react-table@9.2.4   # 指定版本添加
airway js:install                           # 按 js.pkg.json 安装到 app/assets/js/vendor/
airway js:build                             # 打包 app/assets/js 到 app/assets/dist
```

registry 默认为 `https://registry.npmjs.org`；可设置 `AIRWAY_JS_REGISTRY`
（或 `JS_REGISTRY`）使用镜像。整个过程不需要 Node 工具链——CLI 直接与
npm 兼容的 registry 通信，并使用内嵌的 esbuild 打包。

### OpenAPI 文档

```bash
airway openapi:generate                    # 写出 ./openapi.json（OpenAPI 3.2）
airway openapi:generate --out docs/api.json
```

文档覆盖 Gin 引擎上注册的所有路由，运行中的服务器还会在
`GET /openapi.json` 实时提供同一份文档。参见 [OpenAPI 指南](openapi.md)。

### Admin 后台

```bash
airway admin:generate                      # 按 config/admin.toml 生成后台
airway admin:root admin 's3cret'           # 创建管理员账号（role admin）
airway admin:member editor 's3cret'        # 创建非管理员账号（--role=editor|viewer）
```

`admin:generate` 会生成模型、`/api/v1/admin` 下的 JSON API、`/admin` 下的
templ 页面、CRUD island，以及每次运行一份迁移文件。账号存放在 `admin_users`
表（密码经 bcrypt 哈希），请先执行 `db:migrate`。参见
[Admin 后台指南](../../ADMIN.zh-CN.md)。

### 桌面应用

```bash
airway desktop:init [--force]              # 在 ./desktop 生成 Wails v3 目标
```

生成的 `desktop/` 包把同一套 Web 技术栈跑在原生 WebView 窗口里。参见
[桌面应用指南](desktop.md)。

### 静态展示站点与主题

```bash
airway ssg:new [--local[=path]] mysite     # 脚手架静态站点项目
airway ssg:build [--out dist]              # 导出 ssg.go 中定义的站点
airway ssg:serve [--addr 127.0.0.1:3000]   # 本地预览
airway theme:new [--local[=path]] mytheme  # 脚手架主题模块
airway theme:install <module | /path>      # 把主题接入项目的 go.mod
```

参见 [SSG 指南](ssg.md)。

### 应用页面的静态导出

```bash
airway static:build [--out dist]            # 导出 export.go 中注册的页面
airway static:serve [--addr 127.0.0.1:3000] # 本地预览静态页面
```

导出使用与服务器相同的 templ 组件渲染页面，并把前端产物复制到页面旁边。
参见[静态导出指南](static-export.md)。

### Plugin（插件）

生成一个新的 Plugin 模块骨架（用全局安装的 `airway` 即可运行——它只是写文件）：

```bash
airway plugin:new im                              # 目录：im，Plugin 名称：im
airway plugin:new github.com/me/airway-im-plugin  # 名称从路径最后一段推导
```

Plugin 是通过 `plugins.go` 中的空白导入启用的可选功能模块
（参见 [plugin.md](plugin.md)）：

```bash
go run . plugin:list              # 列出已注册的 Plugin 及挂载路径
go run . plugin:install <module>  # 安装 Plugin 的 SQL 迁移、host/ 目录树和 deps/ 目录
go run . plugin:lint              # 检查 Plugin 检出目录的遗留布局问题
```

Plugin 在编译期注册，因此请通过**项目二进制**运行这些命令
（`go run . ...`）：在项目内全局安装的 `airway` 会自动代理，两种形式看到的
都是你项目自己的 Plugin。`plugin:install` 会为复制的迁移分配新的时间戳，
并跳过已安装的文件；之后它们就是普通迁移，由 `db:migrate` / `db:rollback` /
`db:status` 管理。

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

REPL 只能看到编译进当前二进制的模型——项目模型在各自的 init 中通过
`replreg.Register` 自注册。请在你的项目内使用 `go run . repl`（或使用会自动
代理的 `airway repl`）；项目之外全局安装的 `airway` 只能看到框架内置的模型。

## 全局 `airway` 与项目二进制的区别

有些命令依赖编译进当前二进制的代码（模型、Plugin、Go DSL 迁移、路由表）。
在项目内代理让这一点保持透明，但在项目之外仍需区分：

| 命令 | 全局 `airway` | 项目二进制（`go run . ...`） |
| --- | --- | --- |
| `new`、`ssg:new`、`theme:new`、`version`、`help` | 本地执行——生成新项目或描述工具本身 | — |
| `templates:compile` | **本地执行**——必须在项目尚不能编译时可用（刚生成 `.templ` 文件之后） | 可直接运行 |
| `generate`、`server`、`db:*`、`schema:*` | 代理到项目二进制 | 可直接运行 |
| `openapi:generate`、`admin:*`、`desktop:init`、`js:*` | 代理到项目二进制 | 可直接运行 |
| `ssg:build` / `ssg:serve`、`static:*`、`theme:install` | 代理到项目二进制 | 可直接运行 |
| `plugin:new` / `plugin:lint` | 代理到项目二进制 | 可直接运行 |
| `plugin:list` / `plugin:install` | 代理到项目二进制 | **能看到你项目的 Plugin** |
| `upload` | 代理到项目二进制 | 可直接运行 |
| `repl` | 代理到项目二进制 | **能看到你的项目模型** |

在项目之外，全局 CLI 会自己执行命令：遗留 SQL 迁移仍然可用，但 Go DSL 迁移、
Plugin 路由、项目模型和路由表只存在于项目二进制中。

另请参阅：[docs/cli.md](cli.md) 获取完整的脚手架指南和逐步特性示例。
