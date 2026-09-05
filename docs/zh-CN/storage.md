# 文件存储 API 使用文档

Airway 在 `lib/storage` 中提供统一的文件存储层，后端完全通过配置切换：

- `local` — 存储到本地目录（默认 `./data/storage`）
- 云对象存储 — Amazon S3（`s3`）、Cloudflare R2（`r2`）、腾讯云 COS（`cos`）

业务代码只面向 `storage.Storage` 接口编程，本地磁盘和云存储之间切换不需要改动任何代码。

## 1. 配置

所有配置项都是环境变量（本地开发写在 `.env` 中）：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `STORAGE_DRIVER` | `local` | 存储驱动：`local`、`s3`、`r2` 或 `cos` |
| `STORAGE_ROOT` | `./data/storage` | `local` 驱动的存储根目录 |
| `STORAGE_REGION` | — | 区域。`s3` 和 `cos` 必填，endpoint 由它推导；`r2` 默认为 `auto` |
| `STORAGE_ENDPOINT` | 自动推导 | endpoint 覆盖项。`r2` 必填（包含 account ID） |
| `STORAGE_BUCKET` | — | Bucket 名称 |
| `STORAGE_ACCESS_KEY` | — | Access Key ID |
| `STORAGE_SECRET_KEY` | — | Secret Access Key |
| `STORAGE_PUBLIC_URL` | — | 公共访问域名（如 CDN）。设置后文件 URL 直接拼接，不再生成预签名 URL |

云存储后端始终使用 HTTPS。

### 本地存储

```bash
STORAGE_DRIVER="local"
STORAGE_ROOT="./data/storage"
```

### Amazon S3

```bash
STORAGE_DRIVER="s3"
STORAGE_REGION="us-east-1"
STORAGE_BUCKET="my-bucket"
STORAGE_ACCESS_KEY="AKIA..."
STORAGE_SECRET_KEY="..."
```

### 腾讯云 COS

```bash
STORAGE_DRIVER="cos"
STORAGE_REGION="ap-guangzhou"
STORAGE_BUCKET="my-app-1234567890"
STORAGE_ACCESS_KEY="AKID..."
STORAGE_SECRET_KEY="..."
```

### Cloudflare R2

```bash
STORAGE_DRIVER="r2"
STORAGE_ENDPOINT="<account_id>"
STORAGE_BUCKET="my-bucket"
STORAGE_ACCESS_KEY="..."
STORAGE_SECRET_KEY="..."
```

R2 的 `STORAGE_ENDPOINT` 既可以填写 32 位 account ID，也可以填写完整的
`<account_id>.r2.cloudflarestorage.com` endpoint。

存储层在应用启动时自动初始化（见 `main.go`）。配置错误（例如缺少凭证）会导致启动失败并输出错误日志。

## 2. 在 Go 代码中使用

默认实例在启动时已初始化，通过 `storage.Current()` 获取：

```go
import "github.com/daqing/airway/lib/storage"

store := storage.Current()
ctx := context.Background()

// 存储文件（需提供大小与 Content-Type）
err := store.Put(ctx, "docs/report.pdf", storage.Object{
	Reader:      reader,
	Size:        size,
	ContentType: "application/pdf",
})

// 读取文件（调用方负责关闭）
rc, err := store.Get(ctx, "docs/report.pdf")
defer rc.Close()

// 判断是否存在
ok, err := store.Exists(ctx, "docs/report.pdf")

// 获取可下载的 URL
url, err := store.URL(ctx, "docs/report.pdf", 24*time.Hour)

// 删除
err := store.Delete(ctx, "docs/report.pdf")
```

### `Storage` 接口

```go
type Storage interface {
    Put(ctx context.Context, key string, obj Object) error // Object{Reader, Size, ContentType}
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    URL(ctx context.Context, key string, expires time.Duration) (string, error)
}
```

行为说明：

- **key** 是以 `/` 分隔的相对路径，例如 `docs/202609/report.pdf`。空 key、绝对路径和包含 `..` 的 key 会被拒绝，用户输入无法逃逸出存储根目录。
- **URL()** 按后端区分：`local` 返回应用自身的下载路径（`/api/v1/storage/<key>`）；云存储驱动在配置了 `STORAGE_PUBLIC_URL` 时返回 `公共域名/<key>`，否则返回有效期为 `expires` 的预签名 GET URL。当应用配置了 `URL_PREFIX`（例如 `/airway`）时，HTTP 接口返回的本地下载 URL 会自动带上该前缀。
- **Delete** 是幂等的：删除不存在的 key 不会报错。

## 3. HTTP API

`storage_api` 模块提供上传、下载、删除三个 HTTP 接口，挂载在 `/api/v1` 下。

### 上传 — `POST /api/v1/storage`

multipart 表单字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `file` | 是 | 要上传的文件 |
| `dir` | 否 | key 前缀；最终 key 为 `<dir>/<yyyymm>/<随机hex><扩展名>` |

```bash
curl -F "file=@report.pdf" -F "dir=docs" http://127.0.0.1:1900/api/v1/storage
```

响应 `200 OK`：

```json
{
  "key": "docs/202609/8f3a2b1c9d4e5f6a.pdf",
  "url": "/api/v1/storage/docs/202609/8f3a2b1c9d4e5f6a.pdf",
  "size": 12345
}
```

使用云存储驱动时，`url` 为预签名 URL（或配置了 `STORAGE_PUBLIC_URL` 时的 CDN 地址）。

### 下载 — `GET /api/v1/storage/<key>`

```bash
curl -O http://127.0.0.1:1900/api/v1/storage/docs/202609/8f3a2b1c9d4e5f6a.pdf
```

返回文件内容，Content-Type 按扩展名推断；key 不存在时返回 `404`。

### 删除 — `DELETE /api/v1/storage/<key>`

```bash
curl -X DELETE http://127.0.0.1:1900/api/v1/storage/docs/202609/8f3a2b1c9d4e5f6a.pdf
```

响应 `200 OK`：

```json
{"deleted": "docs/202609/8f3a2b1c9d4e5f6a.pdf"}
```

## 4. 测试

- `lib/storage` 的单元测试覆盖本地后端、key 校验和 env 解析，不需要任何云端凭证。
- 云存储后端不在测试中访问真实服务。上线前建议用实际 bucket 做一次真实上传验证。
