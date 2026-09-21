# Admin 管理后台

Airway 可以从一份 TOML 配置文件生成完整、可直接投产的 Admin 管理后台。一条命令即可产出带角色的登录认证、审计日志、带侧边栏导航的仪表盘，以及每个表的类型感知 CRUD——服务端渲染页面加上与 JSON API 交互的 Preact island。

## 快速开始

1. 在 `config/admin.toml` 中描述你的表（见 [配置文件](#配置文件)）。
2. 运行生成器与标准后续步骤：

```bash
airway admin:generate                     # 或: admin:generate --force=table1,table2
airway templates:compile                  # 编译 .templ 视图（等价于 `go generate ./...`）
airway js:build                           # 打包 CRUD island
airway db:migrate                         # 建表
airway admin:user admin@example.com 's3cret' admin   # 创建第一个账号
airway server                             # 访问 /admin
```

3. 在 `/admin/login` 登录后进入仪表盘：每个资源一张卡片、侧边栏导航，以及每个表的完整 CRUD。

## 配置文件

生成器默认读取 `config/admin.toml`；也可以把其他路径作为唯一参数传入。每个顶层表就是一个资源，键名为**单数**形式——SQL 表名使用复数形式（`[post]` → `posts`）。每个字段把列名映射为一个类型：

```toml
[category]
name = "string"
sort_order = "integer"
parent_id = "references:category"    # 自引用（显式指定目标表）

[member.meta]                        # 可选的显示标签（任意语言）
label = "成员管理"
labels = { name = "姓名", email = "邮箱" }

[member]
name = "string"
email = "string"

[post]
title = "string"
body = "text"
views = "integer"
score = "float"
published = "boolean"
published_at = "datetime"
status = "enum:draft,published,archived"
cover = "attachment"
category_id = "references"           # 目标表根据 _id 后缀推断
deleted_at = "datetime"              # 让该表启用软删除
```

### 字段类型

| TOML 类型 | Go 模型字段 | SQL 列 | 表单控件 |
|---|---|---|---|
| `string` | `string` | `VARCHAR(255)` | 文本输入框 |
| `text` | `string` | `TEXT` | 多行文本域 |
| `integer` / `int` | `int64` | `BIGINT` | 数字输入框 |
| `float` | `float64` | `DOUBLE PRECISION` | 数字输入框 |
| `boolean` / `bool` | `bool` | `BOOLEAN` | 复选框 |
| `datetime` | `*time.Time` | `TIMESTAMP` | 日期时间选择器 |
| `enum:a,b,c` | `*string` | `VARCHAR(255)` + `CHECK` | 下拉选择 |
| `references[:table]` | `*int64` | `BIGINT` + 外键 + 索引 | 远程下拉 |
| `attachment` | `string` | `VARCHAR(255)` | 文件上传 |

说明：

- 每张表都会自动带上 `id`、`created_at` 和 `updated_at`，不要在配置里声明。
- 表名和字段名必须是小写 snake_case 标识符。单数键名会复数化为 SQL 表名；
  两张表映射到同一个复数名属于硬错误。
- `enum` 的值以 `*string` 往返，清空下拉框会存入 SQL NULL；生成的 `CHECK`
  约束会拒绝列表之外的值。
- `references` 从 `_id` 后缀推断目标表（`category_id` → `category`），
  也可以显式指定（`references:member`）。支持前向引用；表之间的循环引用
  会被拒绝。references 字段同时会生成 `Relations()` 条目
  （`repo.NewBelongsTo`），可配合 `repo.Preload` 使用。
- 声明 `deleted_at = "datetime"` 即让该表启用**软删除**
  （见 [软删除](#软删除)）。

### `[table.meta]` — 显示标签

可选的 `[table.meta]` 子表可以覆盖生成的显示文案，而不破坏
`field = type` 的格式：

- `label` —— 侧边栏入口、页面标题和仪表盘卡片文案。
- `labels` —— 字段级覆盖，用于表格列头、表单标签和 CSV 表头。

因此面板可以使用 TOML 里书写的任何语言。

## 生成器产出

对 TOML 里的每张表：

- **Model** —— `app/models/<name>.go`，带 `db`/`json` 标签和 REPL 注册；
  可空类型映射为指针（`*string`、`*int64`、`*time.Time`）。
- **Resource** —— `app/api/admin_api/<name>_resource.go`：自注册文件，
  包含 CRUD 动作、列表条件构建器、CSV 渲染器和 OpenAPI 声明。
- **Page** —— `app/views/admin/<plural>_page.templ`，承载该表的 CRUD
  island。
- **Island** —— `app/assets/js/islands/admin-<plural>-crud.tsx`：
  DataTable + 弹窗表单，控件与类型匹配（enum 下拉、远程 references 下拉、
  日期时间选择器、复选框、文件上传）。

每个项目仅生成一次（文件已存在则跳过）：

- `app/models/admin_user.go`、`admin_session.go` —— 登录认证。
- `app/middlewares/admin_auth.go` —— `AdminAuth`、`AdminRequireWrite`、
  `AdminRequireAdmin`。
- `app/api/admin_api/` —— `routes.go`、`registry.go`（共享 helper：分页、
  排序白名单、过滤、CSV 单元格、审计写入）、`auth_action.go`（登录/登出、
  CSRF、限速）、`audit_action.go`、`uploads_action.go`（经
  `storage.Current()` 的附件上传）、`openapi.go`。
- `app/views/admin/` —— `admin.templ`（侧边栏布局）、`index.templ`
  （仪表盘）、`login.templ`、`audit_log.templ`、`types.go`。
- 每次运行一对 migration（`db/migrate/<timestamp>_create_admin_tables.{up,down}.sql`）：
  首次运行包含认证表，之后按外键依赖顺序创建新表，并带外键约束、外键索引
  和 enum `CHECK` 约束。

`config/routes.go` 会自动接线一次（`admin_api.Routes(r)`），失败时回退为
打印手工片段。

## 路由

| 路由 | 访问权限 | 用途 |
|---|---|---|
| `GET /admin` | 所有已登录账号 | 仪表盘 |
| `GET /admin/login` · `POST /admin/login` · `POST /admin/logout` | 公开 | 登录/登出 |
| `GET /admin/<plural>` | 已登录 | 资源页面（CRUD island） |
| `GET /admin/audit-log` | 仅 `admin` 角色 | 审计日志 |
| `GET /api/v1/admin/<plural>` | 已登录 | 列表（JSON），`format=csv` 时返回 CSV |
| `POST /api/v1/admin/<plural>` | editor 及以上 | 创建 |
| `PUT /api/v1/admin/<plural>/:id` | editor 及以上 | 更新 |
| `DELETE /api/v1/admin/<plural>/:id` | editor 及以上 | 删除（或软删除） |
| `POST /api/v1/admin/uploads` | editor 及以上 | 附件上传 |

配置了 `URL_PREFIX` 时，以上路径都在前缀之下提供服务，重定向与链接会
自动带上前缀。

## 认证与账号

账号存放在 `admin_users` 表中，密码经 bcrypt 哈希。创建账号：

```bash
airway admin:user <email> <password> [admin|editor|viewer]
```

默认角色为 `editor`。会话保存在服务端的 `admin_sessions` 表中：登录成功
后签发一个 64 位随机十六进制 token，通过 `airway_admin_session` cookie
下发（HttpOnly、`SameSite=Lax`、7 天有效期、非 `AIRWAY_ENV=local` 环境下
启用 `Secure`）。没有"首次访问创建管理员"页面——账号只在你创建时才存在，
空部署没有任何可被抢注的入口。

## 角色

| 角色 | 读取 | 写入（增/改/删/上传） | 审计日志 |
|---|---|---|---|
| `admin` | ✓ | ✓ | ✓ |
| `editor`（默认） | ✓ | ✓ | — |
| `viewer` | ✓ | —（403） | —（重定向） |

写路由挂在 `AdminRequireWrite` 中间件之后，审计页面挂在
`AdminRequireAdmin` 之后。未知或缺失的角色一律按最小权限处理。

## 安全加固

- **CSRF** —— 登录表单携带随机 double-submit token；只有隐藏字段与
  `airway_admin_csrf` cookie（HttpOnly、`SameSite=Lax`、12 小时有效期）
  一致时，POST 才会被接受。
- **限速** —— 同一 IP 和邮箱组合五次登录失败即锁定十五分钟（进程内
  固定窗口限速器，`lib/ratelimit`）。
- **审计日志** —— 每次创建、更新、删除都会在 `admin_audit_logs` 中记录
  操作账号、资源和记录 id。审计写入失败只记录日志，绝不阻断业务动作。
  admin 可在 `/admin/audit-log` 查看审计记录（最新在前，分页展示）。
- **SQL 安全** —— 列表过滤与排序只接受生成白名单中的标识符；请求值
  始终作为参数绑定。

## 列表 API

`GET /api/v1/admin/<plural>` 全面支持服务端处理：

| 参数 | 含义 |
|---|---|
| `page` | 页码，从 1 开始（默认 1） |
| `page_size` | 每页行数，1–100（默认 20） |
| `q` | 文本搜索，命中所有字符串类字段（PostgreSQL 上为 `ILIKE`） |
| `sort` | 排序列——按资源白名单校验 |
| `order` | `asc`（默认）或 `desc` |
| `<field>=<value>` | 对任意声明字段做类型转换后的精确过滤 |
| `format=csv` | 以 CSV 下载的形式流式返回匹配行，而非 JSON |

响应（JSON）：

```json
{ "code": 0, "data": { "items": [ ... ], "total": 42, "page": 1, "page_size": 20 } }
```

CSV 导出使用显示标签作为表头，经 `encoding/csv` 处理内嵌分隔符的转义，
并中和前导公式字符（`=+-@`），避免电子表格把单元格内容当代码执行。
过滤、`q` 和排序对导出同样生效。

CRUD island 自带配套 UI：搜索框、分页页脚，以及保留当前搜索条件的
Export CSV 链接。

## 软删除

在表里声明 `deleted_at = "datetime"` 会改变它的生命周期：

- 删除变成一次 UPDATE：写入 `deleted_at`（并跳过已删除的行）。
- 所有列表/计数读取都会过滤 `deleted_at IS NULL`。
- 该列保留在模型与 migration 中，但不会出现在表单、参数和 CSV 导出里。

没有该列的表保持硬删除。恢复一行只需一句 SQL/REPL——目前没有恢复 UI。

## 附件

attachment 字段通过 `storage.Current()` 上传——与应用其余部分共用同一套
local/S3/R2/COS 配置——并把生成的 URL 存入列中。island 的文件控件在选中
文件后立即上传，并展示已存文件的链接。

## 重新运行生成器

- 往 TOML 里加表是**纯增量**操作：已有文件绝不会被改写，自注册的
  resource 让侧边栏、仪表盘和路由自动发现新表。
- **`--force`**（或 `--force=table1,table2`）会重写已有表的生成文件——
  model、resource、page 和 island——并丢弃这些文件里的手改内容。它不会
  生成 migration，schema 变更仍需在 `db/migrate/` 里手写。
- 从 TOML 里删除表不会删除已生成的代码；修改字段类型也不会改动已有文件
  或 migration。
- 在角色/审计功能之前生成的项目，下次运行会自动获得一份**升级
  migration**（为 `admin_users` 添加 `role` 列并创建 `admin_audit_logs`）。

## 定制生成产物

`app/` 下的所有内容都属于你。生成的文件带有"可随意编辑"的说明，只有你
显式要求（`--force`）时才会被重新生成，因此可以放心定制：

- 扩展 resource 动作（新增路由、校验、副作用）——`registry.go` 中的共享
  helper（`adminListPaging`、`adminListOrder`、`adminAudit` 等）都是普通
  Go 函数；
- 调整 `app/views/admin/*.templ` 的样式和 `app/assets/css/airway.css` 中的
  `aw-admin-*` 类；
- 收紧每个 resource 文件里的 OpenAPI 声明。

编辑视图或 island 之后别忘了项目惯例：

```bash
airway templates:compile && airway js:build
```

## 当前限制

- 面板内没有用户管理：角色只能通过 `admin:user` 分配。
- 没有软删除恢复 UI（只能通过 SQL/REPL）。
- 限速器是进程内的；多副本部署应在入口处加共享限速。
- 审计日志没有保留策略，也不支持导出。

---

> English version: [ADMIN.md](ADMIN.md)
