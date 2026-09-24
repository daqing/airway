# OpenAPI 接口文档

Airway 可以自动生成 [OpenAPI](https://spec.openapis.org) 3.2 格式的 API
文档，客户端应用（Vue 3、React、SwiftUI 等）可以直接用标准代码生成工具对接。

文档有两种消费方式：

- **文件**：`airway openapi:generate` 生成 `./openapi.json`（本地构建产物，
  已被 git 忽略；路由或声明类型变化后重新生成），代码生成工具直接指向它。
- **实时端点**：运行中的服务在 `GET /openapi.json` 上提供文档，代码生成工具
  可以直接指向已部署的服务。

```bash
airway openapi:generate                    # 生成 ./openapi.json
airway openapi:generate --out docs/api.json
```

输出是确定性的（paths、operation、schema 均排序），变化后重新生成不会产生
无意义的 diff；该文件是本地构建产物，已被 git 忽略。
`info.version` 取自项目 `VERSION` 文件；`servers` 在实时端点模式下由请求
Host 与 `URL_PREFIX` 推导，在 CLI 模式下由 `LISTEN` 与 `URL_PREFIX` 推导。
当应用通过 `URL_PREFIX=/airway` 部署在反向代理之后时，
文档挂载在 `/airway/openapi.json`，其 `servers` URL 自动携带前缀。

## 自动生成的内容

Gin 引擎上注册的每一条路由——包括 [Plugin](plugin.html) 挂载的路由——无需任何
额外工作就会出现在文档里：

- `operationId` 来自 handler 名称（如 `storage_api.UploadAction`）；
- tag 取 `/api/v1` 之外的第一段路径；
- 路由中的路径参数自动识别（gin 的 `:id` / `*key` 归一化为 `{id}` / `{key}`）；
- 默认 `200` 响应按框架信封结构生成（见下文）。

没有 JSON 契约的路由不进入文档。框架自身排除了 `/`、`/ui`、`/assets/*`、
`/ws`、`/openapi.json`，可在 `app/api/openapi_api/doc.go` 中调整。

## 声明更丰富的元数据

自动生成的路由没有请求/响应 schema。要补全它们，在 API 模块下新增
`openapi.go`，在 `init()` 中声明（与 REPL 模型、Plugin 相同的注册机制）：

```go
// app/api/post_api/openapi.go
package post_api

import "github.com/daqing/airway/lib/openapi"

func init() {
	openapi.Get("/api/v1/posts", func(o *openapi.Operation) {
		o.Summary("List posts").Tag("posts").
			Query("page", openapi.Int(), "Page number").
			OK(openapi.List[Post]())
	})

	openapi.Post("/api/v1/posts", func(o *openapi.Operation) {
		o.Summary("Create a post").Tag("posts").
			Body(openapi.Item[CreatePostParams]()).
			OK(openapi.Item[Post]())
	})

	openapi.Put("/api/v1/posts/{id}", func(o *openapi.Operation) {
		o.Summary("Update a post").Tag("posts").
			Path("id", openapi.Int(), "Post id").
			Body(openapi.Item[CreatePostParams]()).
			OK()
	})

	openapi.Delete("/api/v1/posts/{id}", func(o *openapi.Operation) {
		o.Summary("Delete a post").Tag("posts").
			Path("id", openapi.Int(), "Post id").
			OK()
	})
}
```

声明按 method + path 与已注册路由匹配（接受 gin 风格的 `:id` / `*key`，自动
归一化）。声明内容覆盖自动生成的默认值；`airway openapi:generate` 会对没有
匹配到任何路由的声明给出警告。

### 响应信封

`render.OK` 与 `render.Error` 都以 HTTP 200 返回 `{"code", "data", "message"}`
信封，因此 `o.OK(schema)` 会把 200 响应文档化为「成功信封（你的 schema 作为
`data`）」与共享 `Error` 组件的 `oneOf`——这正是客户端需要处理的两种形态。

绕过信封的端点（裸 `c.JSON`、文件下载）直接声明响应：

```go
o.Respond(200, "application/octet-stream", openapi.File()).
	Description("The file contents")
o.Respond(404, "application/json",
	openapi.Obj(map[string]*openapi.Schema{"error": openapi.Str()}))
```

### Schema 助手

| 助手 | 结构 |
| --- | --- |
| `openapi.Item[T]()` | 单个 `T`；命名 struct 按 `json` tag 推断为 `$ref` 组件 |
| `openapi.List[T]()` | `T` 的 JSON 数组 |
| `openapi.Obj(props)` | 由属性表构成的对象 |
| `openapi.Str()` / `Int()` / `Num()` / `Bool()` / `Any()` | 标量 |
| `openapi.File()` | 二进制字符串（`format: binary`） |

推断规则：`json` tag 决定属性名（`-` 排除，`omitempty` 使字段可选），
`time.Time` 映射为 `string` / `date-time`，指针映射为可空类型，嵌套命名
struct 成为被引用的组件。修改 struct 后重新运行 `airway openapi:generate`。

### Operation 选项

`Summary`、`Description`、`Tag`、`ID`（覆盖 operationId）、`Deprecated`、
`Query`、`Path`、`Body`（JSON）、`Form`（form-urlencoded）、`FormUpload`
（multipart，如存储上传）、`Respond`（状态码 + content type）、`Security`
（单操作安全要求）。

## 文档级设置

标题、排除项、tag 元数据与安全方案在 `app/api/openapi_api/doc.go` 中统一
声明：

```go
func init() {
	openapi.DescribeDoc(func(d *openapi.Document) {
		d.Title("My App API").
			Exclude("/legacy/*").
			Tag("posts", "Blog management").
			SecurityScheme("bearer", openapi.Bearer("JWT"))
	})
}
```

各操作用 `o.Security("bearer")` 引用。未声明任何安全方案时，文档根级携带
空的 `security` 列表，明确表示这是公开 API。

## 脚手架

`airway generate api` 与 `airway generate scaffold` 会在路由文件旁生成
`openapi.go` 起步模板，新模块从创建起就进入文档——补充 handler 时同步完善
声明即可。

## 生成客户端

任何支持 OpenAPI 3.x 的生成工具都可以直接消费 `openapi.json`（或实时端点
URL）：

**Vue 3 / React（TypeScript）** —
[openapi-typescript](https://openapi-ts.dev) 配合
[openapi-fetch](https://openapi-fetch.dev)：

```bash
npx openapi-typescript ./openapi.json -o src/api/schema.d.ts
```

**替代方案**：[orval](https://orval.dev) 可以从同一份文档生成带类型的
react-query hooks 或 axios 客户端。

**SwiftUI** — Apple 的
[swift-openapi-generator](https://github.com/apple/swift-openapi-generator)
直接支持 3.2 文档：在 Xcode 工程或 Swift package 中加入该构建插件与
[OpenAPIRuntime](https://swiftpackageindex.com/swift-server/openapi-runtime)，
指向 `openapi.json`，即可为每个 operation 生成带类型的客户端方法。生成的
客户端从文档读取 `servers` URL，因此 `URL_PREFIX` 部署设置会自动生效。
