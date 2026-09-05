package utils

import (
	"fmt"
	"os"
)

const EMPTY_STRING = ""

func GetEnv(key string) (string, error) {
	v := TrimFull(os.Getenv(key))
	if v == EMPTY_STRING {
		return EMPTY_STRING, fmt.Errorf("%s must be set", key)
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

func GetEnvMulti(keys ...string) string {
	for _, key := range keys {
		data, err := GetEnv(key)
		if err == nil {
			return data
		}

		continue
	}

	return EMPTY_STRING
}

func GetEnvOr(firstKey string, secondKey string) string {
	data, err := GetEnv(firstKey)
	if err != nil {
		return GetEnvMust(secondKey)
	}

	return data
}
