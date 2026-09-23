package utils

import (
	"io/fs"
	"os"
	"path/filepath"
)

// CopyFS copies every regular file of fsys into destDir, preserving
// relative paths and creating subdirectories as needed.
func CopyFS(fsys fs.FS, destDir string) error {
	return fs.WalkDir(fsys, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
