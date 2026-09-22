# 静态展示型网站(Showcase Sites)

Airway 可以生成静态展示网站——公司官网、产品落地页、作品集——产物是纯
HTML 目录,可部署到任何静态托管。传统静态站点生成器渲染模板文件,而
Airway 渲染 **Go 代码**:页面在 Go 文件中声明,theme 是提供
[templ](https://templ.guide) 组件和内嵌资源的 Go module,最终导出为一个
自包含的 `dist/` 目录。

展示型站点不需要数据库、Node 工具链,也不需要常驻服务器。

> 背景与设计记录:[SSG.zh-CN.md](../../SSG.zh-CN.md)(English: [SSG.md](../../SSG.md))。

## 快速上手

```bash
airway ssg:new mysite           # 脚手架一个站点项目(或传绝对路径)
cd mysite
airway ssg:build                # 导出静态站点到 dist/
airway ssg:serve                # 在 http://127.0.0.1:3000 预览
```

`ssg:new` 会创建一个最小化的 Go 项目:宿主 `main.go`、站点定义
`ssg.go`,以及 README。它的参数与 `airway new` 相同(module path 或绝对
目录);`--local[=path]` 会把框架和 corporate 主题都指向本地 airway
checkout,方便框架开发。

## 定义站点

项目根目录的 `ssg.go` 是唯一事实来源。它通过 `cmd.SetSSGBuilder` 注册一
个构建函数,`airway ssg:build` 和 `airway ssg:serve` 都基于它运行——与
plugin、REPL 模型相同的编译期注册模式:

```go
func init() {
	cmd.SetSSGBuilder(func() (*ssg.Site, error) {
		s := site.New(ssg.Meta{
			Title:       "Acme Inc",
			Description: "We build ships.",
		})

		s.Use(corporate.Theme)

		s.Page("/", "Home", corporate.Home{
			Heading: "We build ships",
			Tagline: "Family-owned since 1951.",
			Features: []corporate.Feature{
				{Title: "Design", Body: "Hull-first thinking."},
			},
		})

		s.Page("/about", "About", corporate.Content{
			Lead:       "Family-owned since 1951.",
			Paragraphs: []string{"We operate out of Hamburg."},
		})

		return s, nil
	})
}
```

每个 `Page(slug, title, data)` 调用就是一个输出页面。slug 会被规范化为干
净的绝对路径,并决定导出位置:`/` 生成 `dist/index.html`,`/about` 生成
`dist/about/index.html`。主题的 `Assets()` 会拷贝到 `dist/assets/`。

在站点项目内,全局安装的 `airway` 会自动代理到 `go run .`,所以
`airway ssg:build` 与 `go run . ssg:build` 等价。

## 命令

```bash
airway ssg:new [--local[=path]] <module-path | directory>   # 脚手架站点项目
airway ssg:build [--out dist]                               # 导出静态 HTML
airway ssg:serve [--addr 127.0.0.1:3000]                    # 本地预览服务
airway theme:new [--local[=path]] <module-path | path>       # 脚手架新主题 module
airway theme:install <module | /path/to/theme>               # 把主题接入站点项目
```

`ssg:serve` 通过 HTTP 动态渲染页面(资源从主题的内嵌文件系统直接流出),
适合预览;实际部署的是 `ssg:build` 导出的 `dist/`。

## corporate 主题

框架自带 `github.com/daqing/airway/themes/corporate`:响应式布局,头部导航
由站点页面自动生成(当前页高亮),落地页含 hero 和特性卡片栅格,另有纯文
本页和页脚。`Render` 按页面数据类型分发:

| 数据类型 | 渲染内容 |
|---|---|
| `corporate.Home` | 落地页:主标题、副标题、特性卡片栅格 |
| `corporate.Content` | 文本页:导语加正文段落 |
| `templ.Component` | 共享布局内的自定义内容 |
| 其他(含 `nil`) | 仅头部和页脚 |

## 安装主题

第三方主题通过 `airway theme:install` 安装到站点项目——支持已发布的
module 或本地 checkout:

```bash
airway theme:install github.com/daqing/airway-terminal-theme
airway theme:install github.com/example/theme@v1.2.3
airway theme:install ~/src/airway-terminal-theme   # 本地目录:添加 replace
```

该命令会把主题接入项目的 go.mod(已发布模块写 `require`;本地目录写
`replace` + `require`,且该目录必须是依赖 `github.com/daqing/airway` 的
Go module),解析出主题的包名,并打印在 `ssg.go` 中启用它的两行代码。
主题不会拷贝进项目——与 plugin 一样以 Go module 依赖的形式消费。

## 开发主题

用 `airway theme:new` 脚手架一个主题 module——它会从 module 名推导 Go 包
名(`github.com/me/airway-sunset-theme` 推导为包 `sunset`),写出 `Theme`
实现、templ 组件、起步 CSS 和测试,把 templ 版本钉在与框架一致的版本上,
自动生成 templ 视图,并初始化 git 仓库:

```bash
airway theme:new github.com/me/airway-sunset-theme
airway theme:new --local sunset   # 基于当前目录的 airway checkout 开发
```

任何实现 `lib/ssg.Theme` 的 Go 类型都可以作为主题:

```go
type Theme interface {
	Name() string
	Render(ctx *site.Context) templ.Component
	Assets() fs.FS
}
```

- `Render` 接收站点元信息(`ctx.Meta`)、当前页面(`ctx.Page`)以及全部已
  注册页面(`ctx.Pages`,用于导航)。它返回完整的 HTML 文档;CSS 通过
  `Assets()` 提供。
- `Assets` 返回一个 `fs.FS`,导出时拷贝到站点的 `assets/` 目录,并在
  `/assets/` 下被服务——用 `go:embed` 内嵌(若文件在子目录中,先用
  `fs.Sub` 重新定根)。

主题放在独立的 Go module 里(框架仓库中的 `themes/corporate` 是完整示例,
含测试)。站点通过 `s.Use(mytheme.Theme)` 选择主题。

## 部署

`airway ssg:build` 导出自包含的静态目录:每页一个 `index.html` 外加
`assets/`。整个目录上传到任意静态托管即可——GitHub Pages、Netlify、对象
存储 + CDN。新克隆的仓库只需 `go run . ssg:build` 即可重建。
