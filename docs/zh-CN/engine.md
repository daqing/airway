# Airway Engine（引擎）

Engine 是 Airway 的扩展机制，灵感来自 Rails Engine。一个 Engine 是一个自包含的功能模块
——路由、模型、迁移、视图——以**独立 Go module**（通常是独立 git 仓库）的形式分发。
宿主项目通过 `go get` 安装，再加一行 blank import 即可启用。不是每个项目都需要所有功能：
保持应用精简，需要时（IM、管理后台、计费……）再引入对应的 Engine。

## 使用 Engine（宿主项目）

```bash
# 1. 安装模块
go get github.com/example/airway-im-engine

# 2. 启用 —— 在项目根目录 engines.go（package main）中添加 blank import：
#    import (
#        _ "github.com/example/airway-im-engine"
#    )

# 3. 把 Engine 内嵌的 SQL 迁移复制到 db/migrate（如果有的话）
go run . cli engine:install im

# 4. 照常执行迁移
go run . cli db:migrate
```

常用命令：

```bash
go run . cli engine:list            # 列出已注册的 Engine 及其挂载路径
go run . cli engine:install [name]  # 复制 Engine 的 SQL 迁移
```

Engine 注册的路由在它声明的挂载路径下应答（例如 `/api/v1/im`）。选择暴露模型的
Engine，其模型会出现在 `go run . repl` 中，与宿主模型并列。

如果想自己控制某个 Engine 的挂载路径，可以在 `config/routes.go` 中手动挂载：

```go
myengine.Engine.Routes(r.Group("/custom/prefix"))
```

## 开发一个 Engine

Engine 仓库的目录结构与标准 Airway 项目一致：

```
airway-im-engine/
  go.mod                  # module github.com/example/airway-im-engine
                          # require github.com/daqing/airway
  engine.go               # Engine 实现 + init() 注册
  app/
    api/im_api/           # 路由 + action，与宿主项目同样的约定
    models/               # 带 db tag 的模型结构体 + TableName()
    views/                # templ 视图（提交生成的 *_templ.go）
  db/
    migrate/              # 可选：内嵌的 *.up.sql / *.down.sql 迁移文件
```

### 1. 实现并注册 Engine

```go
package imengine

import (
    "github.com/daqing/airway/lib/engine"
    "github.com/example/airway-im-engine/app/api/im_api"
    "github.com/gin-gonic/gin"
)

type IMEngine struct{}

func (IMEngine) Name() string      { return "im" }
func (IMEngine) MountPath() string { return "/api/v1/im" }
func (IMEngine) Routes(r *gin.RouterGroup) {
    im_api.Routes(r)
}

func init() {
    engine.Register(IMEngine{})
}
```

Engine 包的 `init()` 调用 `engine.Register`，因此宿主只需在 `engines.go` 中 blank import
即可启用。名字重复或挂载路径不以 `/` 开头时，`Register` 会 panic（快速失败）。

### 2. 可选能力

实现以下任意接口，框架会自动识别：

```go
// Bootable —— 在 DB/Redis/Storage 就绪后、HTTP 服务启动前调用。
// 这里可以使用 repo.CurrentDB()、storage.Current() 等。
func (IMEngine) Boot() error { ... }

// REPLModelProvider —— 把模型暴露给 `airway repl`。
func (IMEngine) REPLModels() map[string]any {
    return map[string]any{"Message": models.Message{}}
}

// MigrationProvider —— 把 SQL 迁移文件嵌入二进制分发。
//
//go:embed db/migrate
var migrations embed.FS

func (IMEngine) MigrationFS() fs.FS { return migrations }
```

REPL 模型名不能与宿主模型或其他 Engine 的模型重名；冲突时 Engine 的 REPL 模型会被禁用
并输出警告日志。

### 3. 迁移 —— 两种方式

- **Go DSL 迁移**无需安装步骤：在 `init()` 中调用 `lib/migrate/schema` 的
  `schema.RegisterChange`（与宿主项目的 DSL 迁移完全一样），import 后即自动加入全局
  迁移列表。
- **SQL 文件**（`<version>_<name>.up.sql` / `.down.sql`）通过 `MigrationFS()` 内嵌，由
  `airway cli engine:install <name>` 复制到宿主的 `db/migrate/` 并分配新的时间戳。复制后
  就是普通的宿主迁移：`db:migrate`、`db:rollback`、`db:status` 照常工作；重复执行
  `engine:install` 会跳过已安装的文件。

### 4. 视图与 WebSocket

- templ 视图会编译为 Go 代码，Engine 维护自己的 `app/views/` 包并提交生成的
  `*_templ.go` 文件即可，无需特殊处理。
- Engine 可以 import `github.com/daqing/airway/app/websocket`，通过宿主的 Hub 发布实时
  消息。

### 5. Engine 可用的框架包

`lib/` 下的所有包（`repo`、`sql`、`render`、`storage`、`validation`、`utils`……）以及
`app/websocket`，都可以通过 `github.com/daqing/airway/...` 在 Engine 模块中引用。
