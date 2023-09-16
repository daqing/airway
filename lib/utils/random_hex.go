package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomHex(length int) string {
	randomBytes := make([]byte, length/2)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(randomBytes)
}
