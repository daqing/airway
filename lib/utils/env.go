package utils

import (
	"fmt"
	"os"
)

func GetEnv(key string) (string, error) {
	v := TrimFull(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("%s must be set", key)
	}

	return v, nil
}

func GetEnvMust(key string) string {
	val, err := GetEnv(key)
	if err != nil {
		panic(err)
	}

	return val
}
