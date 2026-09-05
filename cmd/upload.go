package cmd

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/daqing/airway/lib/storage"
)

func runUpload(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printUploadUsage()
		return nil
	}
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: airway cli upload [key] /path/to/file")
	}

	filename := args[len(args)-1]
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open %q: %w", filename, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat %q: %w", filename, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("upload path must be a regular file: %q", filename)
	}

	var key string
	if len(args) == 1 {
		key, err = uploadKey(filename)
	} else {
		key, err = explicitUploadKey(args[0])
	}
	if err != nil {
		return err
	}

	contentType, err := uploadContentType(file, filename)
	if err != nil {
		return err
	}

	store, err := storage.Open(storage.FromEnv())
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}

	obj := storage.Object{
		Reader:      file,
		Size:        info.Size(),
		ContentType: contentType,
	}
	if err := store.Put(context.Background(), key, obj); err != nil {
		return fmt.Errorf("upload %q to %q: %w", filename, key, err)
	}

	fmt.Printf("Uploaded %s to /%s\n", filename, key)
	return nil
}

func printUploadUsage() {
	fmt.Println("usage:")
	fmt.Println("  airway cli upload /path/to/file")
	fmt.Println("  airway cli upload key /path/to/file")
}

func explicitUploadKey(value string) (string, error) {
	key := path.Clean(strings.TrimSpace(value))
	if key == "" || key == "." || strings.HasPrefix(key, "/") || key == ".." || strings.HasPrefix(key, "../") {
		return "", fmt.Errorf("invalid storage key: %q", value)
	}

	return key, nil
}

func uploadKey(filename string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(filename))
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "", fmt.Errorf("upload path must identify a file")
	}

	// Storage keys are root-relative and always use slash separators.
	key := strings.TrimLeft(filepath.ToSlash(cleaned), "/")
	if key == "" || key == ".." || strings.HasPrefix(key, "../") {
		return "", fmt.Errorf("cannot derive storage key from %q", filename)
	}

	return key, nil
}

func uploadContentType(file *os.File, filename string) (string, error) {
	if contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); contentType != "" {
		return contentType, nil
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("detect content type for %q: %w", filename, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind %q: %w", filename, err)
	}
	if n == 0 {
		return "application/octet-stream", nil
	}

	return http.DetectContentType(head[:n]), nil
}
