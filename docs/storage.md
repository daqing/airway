# Storage API Guide

Airway provides a unified file storage layer in `lib/storage`. The backend is selected entirely through configuration:

- `local` — files are stored under a local root directory (default `./data/storage`)
- cloud object storage — Amazon S3 (`s3`), Cloudflare R2 (`r2`) or Tencent COS (`cos`)

Business code only talks to the `storage.Storage` interface; switching between local disk and cloud storage requires no code changes.

## 1. Configuration

All options are environment variables (set them in `.env` for local development):

| Variable | Default | Description |
| --- | --- | --- |
| `STORAGE_DRIVER` | `local` | Backend driver: `local`, `s3`, `r2` or `cos` |
| `STORAGE_ROOT` | `./data/storage` | Root directory for the `local` driver |
| `STORAGE_REGION` | — | Region. Required for `s3` and `cos`; the endpoint is derived from it. `r2` defaults to `auto` |
| `STORAGE_ENDPOINT` | derived | Optional endpoint override. Required for `r2` (it carries the account ID) |
| `STORAGE_BUCKET` | — | Bucket name |
| `STORAGE_ACCESS_KEY` | — | Access key ID |
| `STORAGE_SECRET_KEY` | — | Secret access key |
| `STORAGE_PUBLIC_URL` | — | Public base URL (e.g. a CDN domain). When set, file URLs are built from it instead of presigned URLs |

Cloud backends always use HTTPS.

### Local backend

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

### Tencent COS

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
STORAGE_ENDPOINT="<account_id>.r2.cloudflarestorage.com"
STORAGE_BUCKET="my-bucket"
STORAGE_ACCESS_KEY="..."
STORAGE_SECRET_KEY="..."
```

The storage layer is initialized automatically at application boot (`main.go`). A bad configuration (e.g. missing credentials) aborts startup with an error.

## 2. Usage in Go code

The default instance is set up at boot; access it with `storage.Current()`:

```go
import "github.com/daqing/airway/lib/storage"

store := storage.Current()
ctx := context.Background()

// Store a file
err := store.Put(ctx, "docs/report.pdf", reader, size, "application/pdf")

// Read a file back (caller must close)
rc, err := store.Get(ctx, "docs/report.pdf")
defer rc.Close()

// Check existence
ok, err := store.Exists(ctx, "docs/report.pdf")

// Get a downloadable URL
url, err := store.URL(ctx, "docs/report.pdf", 24*time.Hour)

// Delete
err := store.Delete(ctx, "docs/report.pdf")
```

### The `Storage` interface

```go
type Storage interface {
    Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    URL(ctx context.Context, key string, expires time.Duration) (string, error)
}
```

Notes on behavior:

- **Keys** are slash-separated relative paths, e.g. `docs/202607/report.pdf`. Empty keys, absolute paths and `..` segments are rejected, so user input cannot escape the storage root.
- **URL()** behaves per backend: `local` returns the app's own download path (`/api/v1/storage/<key>`); cloud backends return `STORAGE_PUBLIC_URL/<key>` when a public URL is configured, otherwise a presigned GET URL valid for `expires`.
- **Delete** is idempotent: deleting a missing key succeeds.

## 3. HTTP API

The `storage_api` module exposes upload, download and delete over HTTP, mounted under `/api/v1`.

### Upload — `POST /api/v1/storage`

Multipart form fields:

| Field | Required | Description |
| --- | --- | --- |
| `file` | yes | The file to upload |
| `dir` | no | Key prefix; the final key is `<dir>/<yyyymm>/<random_hex><ext>` |

```bash
curl -F "file=@report.pdf" -F "dir=docs" http://127.0.0.1:1900/api/v1/storage
```

Response `200 OK`:

```json
{
  "key": "docs/202607/8f3a2b1c9d4e5f6a.pdf",
  "url": "/api/v1/storage/docs/202607/8f3a2b1c9d4e5f6a.pdf",
  "size": 12345
}
```

With a cloud driver, `url` is a presigned URL (or CDN URL when `STORAGE_PUBLIC_URL` is set).

### Download — `GET /api/v1/storage/<key>`

```bash
curl -O http://127.0.0.1:1900/api/v1/storage/docs/202607/8f3a2b1c9d4e5f6a.pdf
```

Returns the file content with a content type inferred from the extension; `404` when the key does not exist.

### Delete — `DELETE /api/v1/storage/<key>`

```bash
curl -X DELETE http://127.0.0.1:1900/api/v1/storage/docs/202607/8f3a2b1c9d4e5f6a.pdf
```

Response `200 OK`:

```json
{"deleted": "docs/202607/8f3a2b1c9d4e5f6a.pdf"}
```

## 4. Testing

- `lib/storage` unit tests cover the local backend, key validation and env parsing; they run without any cloud credentials.
- Cloud backends are not exercised against a real service in tests. Before going to production, verify once against your actual bucket with a real upload.
