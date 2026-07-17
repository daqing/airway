package storage_api

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/daqing/airway/lib/storage"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

// UploadAction uploads a single multipart file field named "file".
// An optional "dir" form field prefixes the generated key.
func UploadAction(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart form field \"file\" is required"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	key := buildKey(c.PostForm("dir"), file.Filename)
	contentType := detectContentType(file.Header.Get("Content-Type"), file.Filename)

	store := storage.Current()

	if err := store.Put(c.Request.Context(), key, src, file.Size, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The file is already stored; a URL failure (e.g. presign error) must not
	// fail the upload, so fall back to the raw key.
	url, err := store.URL(c.Request.Context(), key, 24*time.Hour)
	if err != nil {
		url = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"key":  key,
		"url":  url,
		"size": file.Size,
	})
}

// DownloadAction downloads the file identified by the wildcard key.
func DownloadAction(c *gin.Context) {
	key := strings.TrimPrefix(c.Param("key"), "/")
	if key == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	ctx := c.Request.Context()
	store := storage.Current()

	exists, err := store.Exists(ctx, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	r, err := store.Get(ctx, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer r.Close()

	c.Header("Content-Type", detectContentType("", key))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, r)
}

// DeleteAction removes the file identified by the wildcard key.
func DeleteAction(c *gin.Context) {
	key := strings.TrimPrefix(c.Param("key"), "/")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	if err := storage.Current().Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": key})
}

// buildKey generates "dir/yyyymm/<random_hex><ext>"; dir is optional.
func buildKey(dir, filename string) string {
	name := time.Now().Format("200601") + "/" + utils.RandomHex(16) + strings.ToLower(filepath.Ext(filename))

	dir = strings.Trim(strings.TrimSpace(dir), "/")
	if dir == "" {
		return name
	}

	return dir + "/" + name
}

func detectContentType(header, filename string) string {
	if header != "" {
		return header
	}

	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); ct != "" {
		return ct
	}

	return "application/octet-stream"
}
