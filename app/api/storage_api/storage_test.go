package storage_api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daqing/airway/lib/storage"
	"github.com/gin-gonic/gin"
)

const apiPrefix = "/api/v1/storage"

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	if _, err := storage.Setup(storage.Config{Driver: storage.DriverLocal, Root: t.TempDir()}); err != nil {
		t.Fatalf("storage.Setup: %v", err)
	}

	gin.SetMode(gin.TestMode)

	r := gin.New()
	Routes(r.Group("/api/v1"))

	return r
}

func doRequest(r *gin.Engine, method, target string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	var reader *bytes.Buffer
	if body != nil {
		reader = body
	} else {
		reader = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	return resp
}

func uploadFile(t *testing.T, r *gin.Engine, filename, content string) map[string]any {
	t.Helper()

	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}

	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	resp := doRequest(r, http.MethodPost, apiPrefix, &buf, w.FormDataContentType())
	if resp.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	return result
}

func TestUploadDownloadDeleteFlow(t *testing.T) {
	r := setupTestRouter(t)

	uploaded := uploadFile(t, r, "hello.txt", "hello storage")

	key, _ := uploaded["key"].(string)
	if key == "" {
		t.Fatalf("upload response missing key: %#v", uploaded)
	}

	if url, _ := uploaded["url"].(string); url != apiPrefix+"/"+key {
		t.Fatalf("unexpected url: %q", url)
	}

	// download
	resp := doRequest(r, http.MethodGet, apiPrefix+"/"+key, nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("download: expected 200, got %d", resp.Code)
	}

	if resp.Body.String() != "hello storage" {
		t.Fatalf("unexpected download body: %q", resp.Body.String())
	}

	if ct := resp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("unexpected content type: %q", ct)
	}

	// delete
	resp = doRequest(r, http.MethodDelete, apiPrefix+"/"+key, nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	// gone
	resp = doRequest(r, http.MethodGet, apiPrefix+"/"+key, nil, "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("download after delete: expected 404, got %d", resp.Code)
	}
}

func TestUploadWithoutFile(t *testing.T) {
	r := setupTestRouter(t)

	resp := doRequest(r, http.MethodPost, apiPrefix, &bytes.Buffer{}, "multipart/form-data; boundary=x")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
