# 静态导出（应用页面上 CDN）

Airway 可以把服务端渲染的应用页面导出为自包含的静态目录：纯 `index.html`
文件加提交在仓库里的前端产物，直接上传到任何静态托管或 CDN。页面 HTML 由
服务端所用的同一批 templ 组件渲染；交互岛屿照常工作——island 运行时本就
是从每页内嵌的 props 客户端挂载的（见[前端指南](frontend.md)）。

这是[静态展示站点](ssg.md)在应用页面一侧的对应能力：`ssg:build` 导出主题
驱动的站点项目，`static:build` 导出 Airway 应用自身的页面。
English version: [docs/static-export.md](../static-export.md).

## 快速上手

```bash
go run . static:build          # 自动重建前端产物、渲染已注册页面、拷贝资源到 ./dist
airway static:serve            # 在 http://127.0.0.1:3000 预览
```

项目内全局安装的 `airway` 会自动代理到 `go run .`，所以
`airway static:build` 与 `go run . static:build` 等价。

导出有两类输入，刷新规则各自不同：

- **前端源码**（`app/assets/js`）在每次导出前自动重建——无需手动
  `js:build`。没有 vendor 目录（未跑 `js:install`）时回退到提交的产物并
  警告；真实构建错误会终止导出。
- **templ 视图**（`app/views/**.templ`）同样自动刷新：视图比生成文件新时，
  命令会先重新生成再重执行自身，因此单条 `static:build` 就能导出改动后的
  视图。生成失败会终止导出，而不是静默交付旧页面。

## 注册页面

新项目自带一份开箱即用的 `export.go`（在项目根目录，导出欢迎页），所以
`static:build` 无需任何配置即可运行。页面在项目根包的 init 函数里通过
`cmd.SetStaticPages` 注册——与 plugin、REPL 模型、ssg 站点构建器相同的
编译期注册模式：

```go
package main

import (
	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/app/views/posts"
	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/static"
)

func init() {
	cmd.SetStaticPages(
		static.Page{Slug: "/", Component: home.Index(3)},
		static.Page{Slug: "/about", Component: posts.AboutPage()},
	)
}
```

每个页面导出为 `<out>/<slug>/index.html`（`/` 生成 `index.html`，`/about`
生成 `about/index.html`）。slug 按 `ssg:build` 相同的规则规范化为干净的绝
对路径，重复注册会报错。

### 脚手架资源

`airway generate scaffold post title:string` 会替你写好注册文件
`export_posts.go`——每个 scaffold 资源一个文件，多次 scaffold 无需合并。
列表页是静态的；详情页在导出时通过 `cmd.SetStaticPagesProvider` 从数据库
枚举：

```go
func init() {
	cmd.SetStaticPages(
		static.Page{Slug: "/posts", Component: posts.Index()},
	)

	// 详情页在导出时枚举——每行一页。
	// 运行 static:build/static:serve 时需配置 AIRWAY_DSN。
	cmd.SetStaticPagesProvider(func() ([]static.Page, error) {
		items, err := repo.FindAll[models.Post]()
		if err != nil {
			return nil, fmt.Errorf("enumerate posts: %w", err)
		}

		pages := make([]static.Page, 0, len(items))
		for _, item := range items {
			pages = append(pages, static.Page{
				Slug:      fmt.Sprintf("/posts/%d", item.ID),
				Component: posts.Show(item),
			})
		}
		return pages, nil
	})
}
```

scaffold 同时生成详情页本身（`app/views/posts/show.templ`，由
`GET /posts/:id` 渲染）：完整 HTML，行内容直接烤进页面——这正是搜索引擎
看到的内容。provider 只在 `static:build`/`static:serve` 收集页面时运行
（进程启动时绝不运行）；`static:build` 在 provider 需要数据库时用
`AIRWAY_DSN`/`DSN` 建立连接；没有数据库页面的项目无需 DSN 也能导出。

## 工作原理

- **离线渲染。** 页面 `Component` 用 `context.Background()` 渲染——无需
  HTTP 请求、无需数据库。注册时组件捕获到的数据就是导出的内容。
- **岛屿天然兼容。** `assets.Island(...)` 渲染空挂载点加内联的
  `<script type="application/json">` props 块；`app.js` 里的 Preact 运行时
  在客户端挂载。没有 SSR hydration 步骤，静态 HTML 与挂载后的岛屿不可能
  不一致。
- **资源随页面一起导出。** `static:build` 把 `app/assets/dist/` 拷贝到
  `<out>/assets/`，与布局输出的 `/assets/app.js?v=…` 路径对应（`?v=` 哈希
  来自构建 manifest）。
- **环境被固定。** 命令在渲染前强制 `AIRWAY_ENV=production`，开发者的本
  地 `.env`（常为 local）不可能产出会 404 的开发资源路径（内存
  livereload bundle）。

`static:serve` 通过 HTTP 动态渲染同样的页面并从 `/assets/` 服务
`app/assets/dist`，不先导出也能完整预览交互。

## 什么能导出、什么不能

| 页面类型 | 静态导出 | 说明 |
| --- | --- | --- |
| 静态内容页（落地页、文档、营销页） | ✅ | 完整 HTML，SEO 友好 |
| props 构建期已知的岛屿 | ✅ | props 烤进 JSON 块 |
| 请求期查数据库的页面 | ⚠️ | 导出的是构建时快照：注册期枚举数据行，或内容变化时重新构建部署 |
| 动态路由（`/posts/:id`） | ⚠️ | 在 `init()` 里循环为每个实例注册一页 |
| POST 表单、WebSocket、auth、`useApiQuery` 请求 | ❌ | 需要活服务器；CDN 只托管外壳 |

经验法则：内容放 templ 层（搜索引擎看到的就是它——岛屿挂载点在 HTML 里
是空的），岛屿只包交互部分。

## 部署

`airway static:build` 导出自包含目录：

```
dist/
  index.html            # 每个注册页面一个 index.html
  about/index.html
  assets/               # app/assets/dist 原样拷贝
    app.js  app.css  manifest.json  *.map
```

整个目录上传到任意静态托管——GitHub Pages、Netlify、对象存储 + CDN。两
个注意点：

- **子路径部署**：构建时设置 `URL_PREFIX`（或 `AIRWAY_URL_PREFIX`）让资源
  URL 带上前缀，并把目录上传到对应路径下。
- **templ 视图有改动**：会自动处理——命令检测到过期视图后会重新生成并
  重执行自身，导出前无需手动跑 `templates:compile`。

新克隆的仓库用 `go run . static:build` 一条命令即可重建。
