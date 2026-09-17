package jspkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTarball unpacks an npm package tarball (gzipped tar, rooted at
// "package/") into dest.
func ExtractTarball(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("tarball is not valid gzip: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		name := filepath.Clean(hdr.Name)
		if name != "package" && !strings.HasPrefix(name, "package"+string(os.PathSeparator)) {
			continue
		}
		target := filepath.Join(dest, strings.TrimPrefix(name, "package"))

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
}

// CheckIntegrity verifies data against an npm SRI string ("sha512-BASE64").
// An empty integrity string passes (registry documents without one).
func CheckIntegrity(data []byte, integrity string) error {
	if integrity == "" {
		return nil
	}
	algo, encoded, ok := strings.Cut(integrity, "-")
	if !ok {
		return fmt.Errorf("malformed integrity %q", integrity)
	}
	if algo != "sha512" {
		return fmt.Errorf("unsupported integrity algorithm %q", algo)
	}
	sum := sha512.Sum512(data)
	actual := base64.StdEncoding.EncodeToString(sum[:])
	if actual != encoded {
		return fmt.Errorf("integrity mismatch: want sha512-%s, got sha512-%s", encoded, actual)
	}
	return nil
}

// SRIFor returns the npm integrity string for data (sha512), used in tests
// to construct registry fixtures.
func SRIFor(data []byte) string {
	sum := sha512.Sum512(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
}

// installedVersion reports the version of a package already unpacked in
// dir, or "" when it is not installed.
func installedVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Version
}

// ExtractTarballBytes is a convenience wrapper for in-memory tarballs.
func ExtractTarballBytes(data []byte, dest string) error {
	return ExtractTarball(bytes.NewReader(data), dest)
}
