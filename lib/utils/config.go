package utils

import "os"

const LOCAL_ENV = "local"

type appConfig struct {
	IsLocal bool
	Env     string
}

func AppConfig() *appConfig {
	env := TrimFull(os.Getenv("AIRWAY_ENV"))

	return &appConfig{
		IsLocal: env == LOCAL_ENV,
		Env:     env,
	}
}
