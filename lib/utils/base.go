package utils

const API_PREFIX = "airway_"

func GenerateApiToken() string {
	return API_PREFIX + RandomHex(20)
}

const ROOT_PATH = "/"

// const EMPTY_PATH = ""

// PathPrefix returns the path for an app,
// regarding the env variable `AIRWAY_MULTI_APP`
// if AIRWAY_MULTI_APP is "1", then call this function
// with "blog" will return "/blog", otherwise returns
// root path
func PathPrefix(app string) string {
	multiApp, _ := GetEnv("AIRWAY_MULTI_APP")

	if multiApp == "1" {
		return ROOT_PATH + app
	}

	return ROOT_PATH
}
