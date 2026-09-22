# README 全量更新计划(v0.5.0 → v0.15.0 + 当前分支)

> 目标:让 `README.md`(英文)与 `docs/zh-CN/README.md`(中文)完整反映框架的
> 当前实现。调研范围:v0.5.0 → v0.15.0 全部 24 个 tag(110 个提交),外加
> `feat/ssg` 分支上已提交未发版的 SSG/主题功能。
>
> 状态:**已执行(2026-09-22)**——README.md 与 docs/zh-CN/README.md 已按本计划重写,
> 验证清单全过。用户已拍板:§7 采纳推荐方案(短节概述+链接)、CLI 全量分组列出、不放版本历史。
> 注:原 PLAN.md(admin 面板计划)已被本计划覆盖,原内容见 git 历史
> (提交 `0dfe08a`)。

## 1. 版本变更清单(调研结论)

| 版本 | 主要变更 | 对 README 的影响 |
|---|---|---|
| v0.6.0–0.6.2 | templ 服务端渲染视图;`URL_PREFIX` 反向代理子路径;框架首页 | 视图章节已有;URL_PREFIX 已有 |
| v0.7.0–0.7.4 | 插件机制(初名 engine,后改名 plugin);CLI 独立为 go-installable 单二进制;`--version`/VERSION 文件;`airway new` 播种 .env | Features 已有 plugin 一条;CLI 清单**缺** plugin:* 命令 |
| v0.8.0–0.8.4 | plugin:install 自包含化(deps/ 目录);`airway new` 支持绝对路径、git init、版本 pin | 细节不需进 README;CLI 行为已提 |
| v0.9.0–0.9.3 | **前端栈**:js:add/js:install(免 Node)、js:build(内嵌 esbuild + livereload)、Preact islands、airway-ui 组件库 + /ui;generate island/scaffold;plugin SQL 迁移安装、host/ 树镜像、全局 CLI 项目代理 | 已有完整章节 ✓ |
| v0.10.0–0.10.2 | plugin:lint;plugin install/ 布局;**OpenAPI 3.2 文档生成**(`openapi:generate` + GET /openapi.json);进程环境优先于 .env | **完全缺失**:Features、CLI、HTTP endpoints、Configuration 四处都要补 |
| v0.11.0 | 事务绑定仓储助手(`repo.WithTx`/`*With` 变体) | 已有 Transactions 章节 ✓ |
| v0.12.0–0.12.1 | Go 1.27.1;plugin 支持文件系统路径 | README **未声明 Go 版本要求** → 补 |
| v0.13.0–0.13.2 | Docker 镜像国内 mirror(daocloud/goproxy.cn);脚手架后自动装前端依赖;**SKIP LOCKED 原生支持**(`ForUpdateSkipLocked`);全仓测试覆盖 | SKIP LOCKED 与 mirror **未提** → 补 Features/Deployment |
| v0.14.0 | **Admin 后台生成器**:admin:generate(TOML 驱动)+ admin:root/admin:member、角色、审计日志、服务端分页 | 仅文档列表有链接;**Features、CLI 清单缺失** → 补章节 |
| v0.15.0 | **Wails v3 桌面导出**:`desktop:init`,lib/boot 程序化启动,三平台打包 | CLI 清单已有;**Features 缺失**,docs/desktop.md 未入文档列表 → 补 |
| feat/ssg(≥v0.15.0) | **SSG + 主题**:`ssg:new/build/serve`、`theme:new/install`、`lib/ssg`、themes/corporate | CLI 清单已补;**Features 缺失** → 补章节 |

## 2. 现有 README 的问题清单

1. **Features 列表缺 5 项**:OpenAPI 文档生成、Admin 后台生成、桌面导出、SSG/主题、SKIP LOCKED(SQL builder 条目内补一句)。
2. **CLI 命令清单不全**:缺 `openapi:generate`、`admin:generate|admin:root|admin:member`、`plugin:new|list|install|lint`、`theme:new|install`、`templates:compile`、`repl`。
3. **HTTP endpoints 表缺** `GET /openapi.json`(live 文档端点)。
4. **Configuration 缺**两类信息:进程环境优先于 .env 的显式说明;`AIRWAY_DB_DSN`/`AIRWAY_PG` 兼容别名(CLI 段落有提,配置表没有)。
5. **文档列表(顶部 + 底部"Guides")重复且不一致**:底部缺 admin/ssg/desktop/openapi;两处应合并为一处。
6. **未声明 Go 版本要求**(go 1.27.1)。
7. **Deployment 未提** Dockerfile 的国内镜像(goproxy.cn / daocloud)。
8. **docs/zh-CN/README.md(428 行)与英文版(578 行)结构已分叉**,不是镜像;需按新版英文全量重写对齐。

## 3. 新 README.md 结构(英文,约 650 行)

| # | 章节 | 处理 | 说明 |
|---|---|---|---|
| 1 | 文档索引(顶部列表) | 重写 | 合并现有顶部+底部 Guides;每行成对(en/zh);补 ssg、admin、desktop、openapi |
| 2 | What Airway is | 微调 | lib/ 清单补 openapi、ssg |
| 3 | Features | 扩充 | +OpenAPI、+Admin 生成、+桌面导出、+SSG/主题、SQL builder 条目加 SKIP LOCKED |
| 4 | Quick start | 微调 | 加 Go 1.27.1 要求;其余保留 |
| 5 | Configuration | 补 | 环境优先级句;DSN 兼容别名移入表格 |
| 6 | HTTP endpoints | 补一行 | `GET /openapi.json` |
| 7 | HTML views / islands / Frontend strategy | 保留 | 无实质变化 |
| 8 | **API documentation (OpenAPI)** | 新增 | 短节:handler 推导 operationId、openapi.go 富化、命令 + live 端点;详见 docs/openapi.md |
| 9 | **Admin panel** | 新增(短节) | 一段能力概述 + 3 条命令 + 链接 ADMIN.md(细节留专文档) |
| 10 | **Desktop apps** | 新增(短节) | 一段模式说明 + desktop:init + 链接 docs/desktop.md / WAILS.md |
| 11 | **Static showcase sites (SSG)** | 新增(短节) | 一段 + 命令 + 链接 SSG.md / docs/ssg.md |
| 12 | CLI | 重写命令清单 | 补全全部命令(与 `airway help` 输出逐条核对) |
| 13 | Repository API / REPL / File storage | 保留 | 现状准确 |
| 14 | Deployment | 补 | mirror 说明一句 |
| 15 | (删)底部 Guides | 删 | 并入顶部索引 |

原则:**README 保持"总览 + 快速上手"定位**,admin/desktop/ssg 各给一小节(≤10 行)+ 专文档链接,细节不搬进 README。

## 4. 中文版策略

`docs/zh-CN/README.md` 按"英文版的完整镜像"标准**全量重写**(当前 428 行、结构分叉):
章节一一对应、代码块与命令完全一致、叙述用地道中文;文首保留 English README 互链。

## 5. 执行步骤

1. 按 §3 写新版 `README.md`(英文)。
2. 按 §4 全量重写 `docs/zh-CN/README.md`。
3. 清理:删除 AGENTS.md 中悬空的 "(see PLAN.md Phase 3)" 引用(前端计划已移除)。
4. 验证(§6)后交用户 review,不主动 commit。

## 6. 验证清单

- [ ] `airway help` 输出与 README CLI 清单逐条一致(以命令实测为准)
- [ ] README 中出现的每个相对链接指向存在的文件(脚本校验)
- [ ] 配置表与 `.env.example` 对照无遗漏
- [ ] HTTP endpoints 与 `config/routes.go` + openapi 路由对照
- [ ] 中文版章节结构与英文版一一对应(脚本比对标题树)
- [ ] `go build ./... && go test ./...` 不受影响(纯文档改动)

## 7. 开放问题(review 时请拍板)

1. Admin / Desktop / SSG 三个短节的详略:按"一段概述 + 命令 + 链接"处理是否合适?(替代方案:Admin 展开成完整章节,但会与 ADMIN.md 大量重复)
2. CLI 清单是否全量列出(约 30 行命令),还是按组归并保持紧凑?
3. 版本变更历史是否需要在 README 中体现(如 CHANGELOG 链接)?当前建议:不放,保持 README 面向"当前实现"。
