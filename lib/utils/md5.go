package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
)

func MD5SumFile(file io.Reader) (string, error) {
	hash := md5.New()

	_, err := io.Copy(hash, file)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil

}
