package utils

import "os"

var env = TrimFull(os.Getenv("AIRWAY_ENV"))

const LOCAL_ENV = "local"

type appConfig struct {
	IsLocal bool
	Env     string
}

var defaultConfig = &appConfig{
	IsLocal: env == LOCAL_ENV,
	Env:     env,
}

func AppConfig() *appConfig {
	return defaultConfig
}
