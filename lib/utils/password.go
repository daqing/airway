package utils

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

func EncryptPassword(password string) (string, error) {
	digest, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(digest), err
}

func ComparePassword(digest string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(digest), []byte(password))

	if err != nil {
		log.Println("compare password error:", err)
		return false
	}

	return true
}
