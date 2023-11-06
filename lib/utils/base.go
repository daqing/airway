package utils

import "os"

const API_PREFIX = "airway_"

func GenerateApiToken() string {
	return API_PREFIX + RandomHex(20)
}

const ROOT_PATH = "/"
const EMPTY_PATH = ""

// PathPrefix returns the path for an app,
// regarding the env variable `AIRWAY_MULTI_APP`
// if AIRWAY_MULTI_APP is "1", then call this function
// with "blog" will return "/blog", otherwise returns
// empty string
func PathPrefix(app string) string {
	multiApp := os.Getenv("AIRWAY_MULTI_APP")

	if multiApp == "1" {
		return ROOT_PATH + app
	}

	return EMPTY_PATH
}
