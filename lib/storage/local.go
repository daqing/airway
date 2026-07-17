package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Local stores files under a root directory on disk.
type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}

	return &Local{root: abs}, nil
}

// Root returns the absolute storage root directory.
func (l *Local) Root() string {
	return l.root
}

func (l *Local) path(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("storage key must not be empty")
	}

	cleaned := filepath.Clean(key)
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage key: %q", key)
	}

	return filepath.Join(l.root, cleaned), nil
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// Write to a temp file first so a failed upload never leaves a partial file.
	tmp := path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, path)
}

func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := l.path(key)
	if err != nil {
		return nil, err
	}

	return os.Open(path)
}

func (l *Local) Delete(_ context.Context, key string) error {
	path, err := l.path(key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (l *Local) Exists(_ context.Context, key string) (bool, error) {
	path, err := l.path(key)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// URL returns the app-internal download path served by the storage_api routes.
func (l *Local) URL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "/api/v1/storage/" + filepath.ToSlash(filepath.Clean(key)), nil
}
