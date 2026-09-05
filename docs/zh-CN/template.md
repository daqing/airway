# Airway 视图使用指南（templ）

Airway 中服务端渲染的 HTML 页面使用 [templ](https://templ.guide/) 编写。
`.templ` 文件看起来就是普通的 HTML，只是穿插了 Go 表达式；在构建时它会被
**编译成普通的 Go 函数**，返回一个类型化的 `templ.Component`。请求时不存在
模板解析的过程——页面就是高效的 Go 代码。

本指南介绍视图放在哪里、如何编写视图，以及 action 如何渲染它们。

## 视图存放位置

视图放在 `app/views/` 下，每个 API 模块一个目录。目录名是去掉 `_api`
后缀后的模块名：

| API 模块 | 视图目录 | 视图文件 |
| --- | --- | --- |
| `home_api` | `app/views/home/` | `index.templ` |
| `post_api` | `app/views/post/` | `show.templ`、`index.templ`、… |
| （共享） | `app/views/layouts/` | `base.templ` |

每个目录是各自独立的 Go 包，包名即目录名。一个包内可以放任意多个视图文件。

### 生成文件会被提交

生成器会把每个 `*.templ` 文件转换为同目录下的 `*_templ.go` 文件（例如
`app/views/home/index.templ` → `app/views/home/index_templ.go`）。这些生成
文件**会提交到版本库**，因此 `go build` 和 `go test` 都不需要安装 templ CLI
（templ 已在 `go.mod` 中通过 Go tool 固定版本）。修改 `.templ` 文件后请重新
生成，并保持输出同步：

```bash
go generate ./...   # 或：just generate
```

## 视图的基本结构

`app/views/home/index.templ`（首页）是最小的可用示例：

```templ
package home

import "github.com/daqing/airway/app/views/layouts"

// Index renders the landing page (GET /).
templ Index() {
	@layouts.Base("Airway") {
		<h1>Hello, Airway!</h1>
		<p>This page is rendered with the templ template engine.</p>
	}
}
```

`templ Index()` 就是一个 component（组件）：从参数映射到 HTML 的“纯函数”。
生成的 Go 代码会导出 `func Index() templ.Component`，也就是 action 实际
调用的对象。

## 共享布局

`app/views/layouts/base.templ` 提供了 HTML 文档的外壳（doctype、head、title、
body）。它声明了一个 `children` 插槽，页面的内容会被注入到这里：

```templ
package layouts

templ Base(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<title>{ title }</title>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}
```

页面通过把内容放在调用的大括号里，把自身标记传给布局（如上所示）。把共享的
标记——导航栏、页脚、脚本——放进 `layouts/`，让每个页面都套用这个外壳，
这样就不用重复写整份文档骨架了。

## 在 action 中渲染视图

页面响应使用 `lib/render` 的 HTML 辅助函数，它是 JSON 辅助函数
（`render.OK`、`render.Error`）在页面场景下的对应物：

```go
package home_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/lib/render"
)

func IndexAction(c *gin.Context) {
	render.HTML(c, home.Index())
}
```

可用的辅助函数：

- `render.HTML(c, component)` —— 以状态码 `200` 和 `Content-Type: text/html`
  发送页面。
- `render.HTMLStatus(c, status, component)` —— 同上，但显式指定状态码
  （例如 `http.StatusNotFound`）。

组件会先渲染到缓冲区，因此渲染失败时返回 `500`，而不是半截响应。

## 向视图传递数据

组件通过参数接收所需的数据。展示单条记录的页面可能长这样：

```templ
package post

import "github.com/daqing/airway/app/views/layouts"

type Post struct {
	Title     string
	Body      string
	Published bool
	Tags      []string
}

templ Show(post Post) {
	@layouts.Base(post.Title) {
		<h1>{ post.Title }</h1>
		<article>{ post.Body }</article>
	}
}
```

视图引用的类型可以在同一个 `.templ` 文件的头部声明（这里的 Go 代码都会被
写入生成文件）。action 负责根据 model 或 service 的结果构造参数：

```go
func ShowAction(c *gin.Context) {
	post, err := services.FindPostByID(...)   // 你的 service 层返回什么就传什么
	if err != nil {
		render.Error(c, err)
		return
	}

	render.HTML(c, post_view.Show(post))
}
```

保持组件**纯净**：数据通过参数传入，不要在模板内部查询数据库或访问请求。
Handler 负责取数据并渲染，模板只负责格式化。

### 插值、流程控制与原始 HTML

大括号 `{ }` 内可以是任意 Go 表达式。值默认会被 HTML 转义，因此用户输入
无法注入标记。流程控制就是普通 Go——`if`、`else`、`for`、`switch`，使用
Go 的大括号：

```templ
<h1>{ post.Title }</h1>

if post.Published {
	<p>Published</p>
}

<ul>
	for _, tag := range post.Tags {
		<li><a href={ "/tags/" + tag }>{ tag }</a></li>
	}
</ul>
```

如果确实需要输出可信任的 HTML（例如已经消毒过的 Markdown 渲染结果），
可以用 `templ.Raw` 包一层：

```templ
<article>{ templ.Raw(post.HTMLBody) }</article>
```

优先使用带转义的 `{ expr }` 写法；只有在你完全信任的内容上才用 `templ.Raw`。

## 实战示例：新增一个页面

为 `post_api` 模块新增一个 HTML `show` 页面。

1. 创建 `app/views/post/show.templ`（内容见上面的示例）。
2. 重新生成 Go 代码：

   ```bash
   go generate ./...
   ```

   这会生成 `app/views/post/show_templ.go`，其中包含 `func Show(...)
   templ.Component`。

3. 在 action 里渲染它（见上面的 `ShowAction` 示例）。
4. 在 `config/routes.go` 或模块自己的 `routes.go` 中注册路由。

## 工作流说明

- **每次编辑后都要重新生成。** 修改 `.templ` 文件后执行
  `go generate ./...`（或 `just generate`），并把生成的 `*_templ.go`
  连同 `.templ` 源文件一起提交。
- **错误在编译期暴露。** 因为模板会编译成 Go，拼写错误和类型不匹配会在
  构建时失败，而不是在请求时。
- **编辑器支持。** templ 自带 LSP（`go tool templ lsp`）和 IDE 扩展；
  用 `go tool templ fmt` 格式化 `.templ` 文件，或在 CI 中用 `go tool
  templ fmt -fail .` 强制检查格式。
- **安全性。** 字符串插值默认会转义；构造 URL 和属性时请用表达式，而不要
  手工拼接 HTML。只有对内容完全信任时才使用 `templ.Raw`。

## 更多资料

- 内置示例：`app/views/home/` + `app/api/home_api/index_action.go`
  （`GET /`）。
- 渲染辅助函数：`lib/render/html.go`。
- [templ 官方文档](https://templ.guide/)：完整的模板语法。
