# Airway Plugin（插件）

Plugin 是 Airway 的扩展机制，命名方式与 WordPress 的插件一致。一个 Plugin 是一个自包含
的功能模块——路由、模型、迁移、视图——以**独立 Go module**（通常是独立 git 仓库）的
形式分发。宿主项目通过 `go get` 安装，再加一行 blank import 即可启用。不是每个项目都
需要所有功能：保持应用精简，需要时（IM、管理后台、计费……）再引入对应的 Plugin。

## 使用 Plugin（宿主项目）

```bash
# 1. 启用 Plugin，把它内嵌的 SQL 迁移复制到 db/migrate，
#    并把它的 deps/ 目录合并到项目的 deps/ 目录。
#    这一条命令会自动执行 `go get`、在 plugins.go 中添加 blank import，
#    并在同一进程中完成迁移和 deps/ 的安装。
go run . plugin:install github.com/example/airway-im-plugin

# 2. 照常执行迁移
go run . db:migrate
```

常用命令：

```bash
go run . plugin:list              # 列出已注册的 Plugin 及其挂载路径
go run . plugin:install <module>  # 启用 Plugin 并安装它的 SQL 迁移和 deps/
```

`plugin:install` 会自动完成启用步骤：如果 Plugin 没有编译进当前二进制，它会向
plugins.go 添加 blank import、执行 `go get <module>`（本地目录则通过 `replace`
指令接入），然后从 Plugin 模块的磁盘目录读取迁移和 deps/ 完成安装——全部在当前
进程内完成，始终使用当前 CLI 的安装逻辑。这些步骤也仍然可以手动完成。

`plugin:list` 只能看到编译进当前二进制的 Plugin。在项目内，全局安装的 `airway`
会自动代理为 `go run .` 执行（stderr 会打印 `proxying to project binary` 提示），
因此列出的就是项目自己启用的 Plugin。

Plugin 注册的路由在它声明的挂载路径下应答（例如 `/api/v1/im`）。选择暴露模型的
Plugin，其模型会出现在 `go run . repl` 中，与宿主模型并列。

如果想自己控制某个 Plugin 的挂载路径，可以在 `config/routes.go` 中手动挂载：

```go
myplugin.Plugin.Routes(r.Group("/custom/prefix"))
```

## 开发一个 Plugin

用 CLI 脚手架一个新的 Plugin 模块（全局安装的 `airway` 即可运行，不涉及编译期注册）：

```bash
airway plugin:new im                              # 目录：im
airway plugin:new github.com/me/airway-im-plugin  # Plugin 名称从路径最后一段推导
```

该命令会生成 `go.mod`、`plugin.go`（Plugin 实现 + `init()` 注册）、
`app/api/<name>_api/` 下的示例 API 模块，以及空的 `app/models/` 和
`host/db/migrate/` 目录，然后自动执行 `go mod tidy`。

Plugin 仓库的目录结构与标准 Airway 项目一致：

```
airway-im-plugin/
  go.mod                  # module github.com/example/airway-im-plugin
                          # require github.com/daqing/airway
  plugin.go               # Plugin 实现 + init() 注册
  app/
    api/im_api/           # 路由 + action，与宿主项目同样的约定
    models/               # 带 db tag 的模型结构体 + TableName()
    views/                # templ 视图（提交生成的 *_templ.go）
  host/                   # plugin:install 时装进宿主项目自身目录树的文件，
    db/migrate/           # 镜像宿主布局；目前仅支持 db/migrate
                          # （可选的 *.up.sql / *.down.sql 迁移文件）
  deps/                   # 可选：plugin:install 时合并进宿主项目 deps/ 目录的
                          # 额外文件（伴生服务、部署配置……），请用 Plugin 名
                          # 作为命名空间（名为 im 的 Plugin 放 deps/im/app/...）
```

### 1. 实现并注册 Plugin

```go
package implugin

import (
    "github.com/daqing/airway/lib/plugin"
    "github.com/example/airway-im-plugin/app/api/im_api"
    "github.com/gin-gonic/gin"
)

type IMPlugin struct{}

func (IMPlugin) Name() string      { return "im" }
func (IMPlugin) MountPath() string { return "/api/v1/im" }
func (IMPlugin) Routes(r *gin.RouterGroup) {
    im_api.Routes(r)
}

func init() {
    plugin.Register(IMPlugin{})
}
```

Plugin 包的 `init()` 调用 `plugin.Register`，因此宿主只需在 `plugins.go` 中 blank import
即可启用。名字重复或挂载路径不以 `/` 开头时，`Register` 会 panic（快速失败）。

### 2. 可选能力

实现以下任意接口，框架会自动识别：

```go
// Bootable —— 在 DB/Redis/Storage 就绪后、HTTP 服务启动前调用。
// 这里可以使用 repo.CurrentDB()、storage.Current() 等。
func (IMPlugin) Boot() error { ... }

// REPLModelProvider —— 把模型暴露给 `airway repl`。
func (IMPlugin) REPLModels() map[string]any {
    return map[string]any{"Message": models.Message{}}
}

// MigrationProvider —— 把 SQL 迁移文件嵌入二进制分发。
//
//go:embed host/db/migrate
var migrations embed.FS

func (IMPlugin) MigrationFS() fs.FS { return migrations }
```

REPL 模型名不能与宿主模型或其他 Plugin 的模型重名；冲突时 Plugin 的 REPL 模型会被禁用
并输出警告日志。

### 3. 迁移 —— 两种方式

- **用 Go 代码写的迁移**无需安装步骤：在 `init()` 中调用 `lib/migrate/schema` 的
  `schema.RegisterChange`（与宿主项目的 Go 代码迁移完全一样），import 后即自动加入全局
  迁移列表。
- **SQL 文件**（`<version>_<name>.up.sql` / `.down.sql`）放在 Plugin 的
  `host/db/migrate/` 目录下，通过 `MigrationFS()` 内嵌或直接从模块磁盘目录读取，
  由 `plugin:install <module>`（在宿主项目中以 `go run . plugin:install <module>`
  运行）复制到宿主的 `db/migrate/` 并分配新的时间戳。复制后
  就是普通的宿主迁移：`db:migrate`、`db:rollback`、`db:status` 照常工作；重复执行
  `plugin:install` 会跳过已安装的文件。

### 4. 用 `deps/` 分发额外的项目文件

Plugin 顶层 `deps/` 目录下的所有内容会被 `plugin:install` 合并到宿主项目的 `deps/`
目录——适合放伴生服务（如独立的 WebSocket gateway）、部署配置等宿主项目需要落在磁盘上的
文件。请用 Plugin 名作为命名空间（名为 `im` 的 Plugin 放在 `deps/im/app/...`），
多个 Plugin 并行安装就不会互相冲突。目标位置已存在的文件会被跳过（绝不覆盖），因此
重复执行 `plugin:install` 是安全的；想用新版 Plugin 刷新某个文件，先删掉已安装的
副本再重新安装。

`airway new` 生成的宿主项目自带空的 `deps/` 目录；目录不存在时安装会自动创建，
老版本 Airway 生成的项目无需任何改动。

两条 Go module 规则决定了你能放什么：

- **`deps/` 内不能有嵌套的 `go.mod`。** module zip 会整体丢弃嵌套 module，真实的
  `go.mod` 永远到不了宿主。请改存为 `go.mod.templ`——安装时会剥离一层 `.templ` 后缀，
  在宿主项目中还原为 `go.mod`。把真实 `go.mod` 与 `.templ` 变体并排放置（便于嵌套
  模块在插件仓库里本地编译）也是允许的：此时安装只取 `.templ` 的内容（裸文件被
  跳过），与 module zip 下载的结果一致；本地目录安装时若发现裸 `go.mod` 与
  `.templ` 内容漂移，安装会立即报错退出。后缀与 templ 模板引擎同名，也为未来安装时
  做动态模板渲染留了余地。`go.sum` 不触发该规则，保持原名即可。
- **目录绝不能叫 `vendor/`。** module zip 会整体丢弃 `vendor/`，这正是约定目录定为
  `deps/` 的原因。

注意：如果 Plugin 同时使用了 templ 视图，`templ generate` 会解析其工作目录下的所有
`.templ` 文件（包括 `deps/` 里的）。请把 generate 指令限定到视图目录
（`//go:generate go tool templ generate -path app/views`)，避免误解析安装模板。

不要在 `deps/` 里提交构建产物（编译出的二进制、缓存等）——它们会被合并进每一个宿主
项目的 `deps/`。

### 5. 视图与 WebSocket

- templ 视图会编译为 Go 代码，Plugin 维护自己的 `app/views/` 包并提交生成的
  `*_templ.go` 文件即可，无需特殊处理。
- Plugin 可以 import `github.com/daqing/airway/app/websocket`，通过宿主的 Hub 发布实时
  消息。

### 6. Plugin 可用的框架包

`lib/` 下的所有包（`repo`、`sql`、`render`、`storage`、`validation`、`utils`……）以及
`app/websocket`，都可以通过 `github.com/daqing/airway/...` 在 Plugin 模块中引用。
