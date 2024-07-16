package utils

import "fmt"

const API_PREFIX = "airway_"

func GenerateApiToken() string {
	return API_PREFIX + RandomHex(20)
}

const SLASH = "/"
const EMPTY_PATH = ""

func FullPath(suffix string) string {
	pwd := GetEnvMust("AIRWAY_ROOT")

	return fmt.Sprintf("%s/%s", pwd, suffix)
}
