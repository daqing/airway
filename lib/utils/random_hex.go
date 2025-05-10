package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomHex generates a random hexadecimal string of the specified length.
func RandomHex(length int) string {
	randomBytes := make([]byte, length/2)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(randomBytes)
}
