package utils

import "fmt"

const API_PREFIX = "airway_"

func GenerateApiToken() string {
	return API_PREFIX + RandomHex(20)
}

const SLASH = "/"
const EMPTY_PATH = ""

// PathPrefix returns the path for an app,
// regarding the env variable `AIRWAY_MULTI_APP`
// if AIRWAY_MULTI_APP is "1", then call this function
// with "blog" will return "/blog", otherwise it returns
// empty path
func PathPrefix(app string) string {
	multiApp, _ := GetEnv("AIRWAY_MULTI_APP")

	if multiApp == "1" {
		return SLASH + app
	}

	return EMPTY_PATH
}

func FullPath(suffix string) string {
	pwd := GetEnvMust("AIRWAY_PWD")

	return fmt.Sprintf("%s/%s", pwd, suffix)
}
