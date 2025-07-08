package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
)

func MD5SumFile(file io.Reader) (string, error) {
	hash := md5.New()

	_, err := io.Copy(hash, file)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil

}

// extName is like: ".png" or ".jpg"
func FilePathFromMD5(md5Hash, extName string) string {
	dir := md5Hash[:2]
	dir2 := md5Hash[2:4]
	fileName := fmt.Sprintf("%s%s", md5Hash[4:], extName)

	return filepath.Join(dir, dir2, fileName)
}
