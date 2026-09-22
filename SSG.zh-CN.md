# 静态展示站点(SSG)——设计记录

Airway 同时也是一个面向展示型网站的静态站点生成器——公司官网、产品落地
页、作品集。站点就是一个普通的 Go 项目:在根包的 `ssg.go` 中面向可替换的
**主题 module** 声明页面,`airway ssg:build` 把它们导出为纯 HTML 目录,可
部署到任何静态托管。页面用与框架服务端渲染相同的
[templ](https://templ.guide) 组件渲染——不需要数据库、Node 工具链,也不
需要常驻服务器。

状态:**已实现并通过端到端验证(2026-09-22)**——`theme:new` →
`theme:install` → `ssg:new` → `ssg:build` → `ssg:serve` 全生命周期在全新
脚手架上跑通。

## 工作方式

```
ssg.go(项目根)                       主题 module(独立 Go module)
  cmd.SetSSGBuilder(init())             实现 lib/ssg.Theme
  ssg.New(meta).Use(theme)              templ 布局 + 页面组件
  s.Page(slug, title, data)             go:embed 内嵌资源(CSS、字体)
        │                                      ▲
        │  airway ssg:build                    │ 以 module 依赖消费
        ▼                                      │ (require,或 replace+require)
  lib/ssg.Build: 渲染每个页面 ────► dist/<slug>/index.html
                  拷贝主题资源 ────► dist/assets/
```

- 项目根包的 `ssg.go` 在 `init()` 中通过 `cmd.SetSSGBuilder` 注册**构建函
  数**——与 plugin、REPL 模型相同的编译期注册模式。`ssg:build` 和
  `ssg:serve` 都基于该构建函数的返回值运行,项目二进制天然携带自己的站点
  代码,`main.go` 无需任何改动。
- `lib/ssg.Build` 把每个注册页面经主题渲染为 `<out>/<slug>/index.html`
  (`/` → `index.html`,`/about` → `about/index.html`),并把主题内嵌资源拷
  贝到 `<out>/assets/`。
- `lib/ssg.Handler` 为 `ssg:serve` 提供同一站点的动态渲染:页面按请求渲
  染,资源从主题内嵌文件系统直接流出。

## 设计决策

1. **静态导出,而非动态托管。** 展示站需要静态托管、SEO 和零运维。templ
   组件可以不经 HTTP 请求直接渲染到任意 `io.Writer`——与 `render.HTML`
   用的是同一个原语——因此导出引擎只是一层薄循环,`ssg:serve` 则复用同一
   渲染做预览。
2. **主题是 Go module,不是模板目录。** 不同于 Hugo/Jekyll 的主题文件夹,
   这里的主题是编译期检查、可单元测试的 Go 代码;分发与版本化直接复用
   module 体系(`theme:install` 在精神上对应 `plugin:install`)。
3. **引擎进核心,主题在外部。** CLI 分发编译在二进制里,插件无法新增命
   令——所以 `ssg:build`/`ssg:serve`/`ssg:new`/`theme:*` 位于 `cmd/`,
   引擎位于 `lib/ssg`。而主题本身对框架零改动,独立的
   `airway-terminal-theme` 证明了这一点。
4. **编译期注册,而非配置。** 从 `init()` 调用 `cmd.SetSSGBuilder` 延续
   Rails 式的"代码优于配置":页面就是 Go 函数调用,构建时不解析任何配置
   文件。

## 实现清单

| 组成 | 位置 |
|---|---|
| Site/Page/Theme 模型、slug 规范化、builder 类型 | `lib/ssg/ssg.go` |
| 静态导出器 + 预览 Handler | `lib/ssg/build.go` |
| `ssg:build`、`ssg:serve`、`cmd.SetSSGBuilder` | `cmd/cli_ssg.go` |
| `ssg:new` 脚手架 | `cmd/cli_ssg_new.go` + `cmd/clitemplate/ssgtemplate/` |
| `theme:new` 脚手架 | `cmd/cli_theme_new.go` + `cmd/clitemplate/themetemplate/` |
| `theme:install` | `cmd/cli_theme.go` |
| 内置参考主题 | `themes/corporate/`(独立 module,replace 指向 checkout) |
| 使用指南 | [docs/ssg.md](docs/ssg.md) / [docs/zh-CN/ssg.md](docs/zh-CN/ssg.md) |

## 验证记录

- **单元测试**(表驱动、无网络):`lib/ssg`(导出布局、预览 Handler、slug
  规范化、注册 panic),`cmd`(ssg 脚手架 local-replace 与版本 pin 两种
  go.mod 形态、主题包名推导、theme install 的 replace/require 与校验错
  误、proxy 豁免)。
- **端到端**(2026-09-22):`ssg:new` → `ssg:build`(slug→路径映射、资源
  拷贝、生成标记)→ `ssg:serve` 经 curl 验证(页面与资源 200,未知路径
  404);`theme:new` 脚手架立即构建、测试全绿;`theme:install` 本地主题后
  切换 `s.Use`,构建产物资源随之切换为已安装主题。
- **门禁**:`go vet ./...`、`go test ./...`(零无测试包)、`gofmt` 干净;主题
  module 为独立 Go module,测试在其各自目录内运行。

> English version: [SSG.md](SSG.md)
> 使用指南(快速上手、命令、页面定义、主题、部署):
> [docs/zh-CN/ssg.md](docs/zh-CN/ssg.md)
