# 文件存储 API 使用文档

Airway 在 `lib/storage` 中提供统一的文件存储层，后端完全通过配置切换：

- `local` — 存储到本地目录（默认 `./data/storage`）
- `s3` — 任意 S3 兼容对象存储：Amazon S3、腾讯云 COS、Cloudflare R2、MinIO 等

业务代码只面向 `storage.Storage` 接口编程，本地磁盘和云存储之间切换不需要改动任何代码。

## 1. 配置

所有配置项都是环境变量（本地开发写在 `.env` 中）：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `STORAGE_DRIVER` | `local` | 存储驱动：`local` 或 `s3` |
| `STORAGE_ROOT` | `./data/storage` | `local` 驱动的存储根目录 |
| `S3_ENDPOINT` | — | S3 兼容服务的 endpoint 主机名 |
| `S3_REGION` | — | 区域；Cloudflare R2 填 `auto` |
| `S3_BUCKET` | — | Bucket 名称 |
| `S3_ACCESS_KEY` | — | Access Key ID |
| `S3_SECRET_KEY` | — | Secret Access Key |
| `S3_USE_SSL` | `true` | 使用 HTTP 协议时设为 `false`（如本地 MinIO） |
| `S3_PATH_STYLE` | `false` | 需要 path-style 寻址的服务设为 `true`（如 MinIO） |
| `S3_PUBLIC_URL` | — | 公共访问域名（如 CDN）。设置后文件 URL 直接拼接，不再生成预签名 URL |

### 本地存储

```bash
STORAGE_DRIVER="local"
STORAGE_ROOT="./data/storage"
```

### Amazon S3

```bash
STORAGE_DRIVER="s3"
S3_ENDPOINT="s3.us-east-1.amazonaws.com"
S3_REGION="us-east-1"
S3_BUCKET="my-bucket"
S3_ACCESS_KEY="AKIA..."
S3_SECRET_KEY="..."
```

### 腾讯云 COS

```bash
STORAGE_DRIVER="s3"
S3_ENDPOINT="cos.ap-guangzhou.myqcloud.com"
S3_REGION="ap-guangzhou"
S3_BUCKET="my-app-1234567890"
S3_ACCESS_KEY="AKID..."
S3_SECRET_KEY="..."
```

### Cloudflare R2

```bash
STORAGE_DRIVER="s3"
S3_ENDPOINT="<account_id>.r2.cloudflarestorage.com"
S3_REGION="auto"
S3_BUCKET="my-bucket"
S3_ACCESS_KEY="..."
S3_SECRET_KEY="..."
```

### MinIO（自建）

```bash
STORAGE_DRIVER="s3"
S3_ENDPOINT="127.0.0.1:9000"
S3_BUCKET="my-bucket"
S3_ACCESS_KEY="minioadmin"
S3_SECRET_KEY="minioadmin"
S3_USE_SSL="false"
S3_PATH_STYLE="true"
```

存储层在应用启动时自动初始化（见 `main.go`）。配置错误（例如缺少 S3 凭证）会导致启动失败并输出错误日志。

## 2. 在 Go 代码中使用

默认实例在启动时已初始化，通过 `storage.Current()` 获取：

```go
import "github.com/daqing/airway/lib/storage"

store := storage.Current()
ctx := context.Background()

// 存储文件
err := store.Put(ctx, "docs/report.pdf", reader, size, "application/pdf")

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
    Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    URL(ctx context.Context, key string, expires time.Duration) (string, error)
}
```

行为说明：

- **key** 是以 `/` 分隔的相对路径，例如 `docs/202607/report.pdf`。空 key、绝对路径和包含 `..` 的 key 会被拒绝，用户输入无法逃逸出存储根目录。
- **URL()** 按后端区分：`local` 返回应用自身的下载路径（`/api/v1/storage/<key>`）；`s3` 在配置了 `S3_PUBLIC_URL` 时返回 `公共域名/<key>`，否则返回有效期为 `expires` 的预签名 GET URL。
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
  "key": "docs/202607/8f3a2b1c9d4e5f6a.pdf",
  "url": "/api/v1/storage/docs/202607/8f3a2b1c9d4e5f6a.pdf",
  "size": 12345
}
```

使用 `s3` 驱动时，`url` 为预签名 URL（或配置了 `S3_PUBLIC_URL` 时的 CDN 地址）。

### 下载 — `GET /api/v1/storage/<key>`

```bash
curl -O http://127.0.0.1:1900/api/v1/storage/docs/202607/8f3a2b1c9d4e5f6a.pdf
```

返回文件内容，Content-Type 按扩展名推断；key 不存在时返回 `404`。

### 删除 — `DELETE /api/v1/storage/<key>`

```bash
curl -X DELETE http://127.0.0.1:1900/api/v1/storage/docs/202607/8f3a2b1c9d4e5f6a.pdf
```

响应 `200 OK`：

```json
{"deleted": "docs/202607/8f3a2b1c9d4e5f6a.pdf"}
```

## 4. 测试

- `lib/storage` 的单元测试覆盖本地后端、key 校验和 env 解析，不需要任何云端凭证。
- S3 后端不在测试中访问真实服务。上线前建议用实际 bucket 做一次真实上传验证。
