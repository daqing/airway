package main

import (
	"log"
	"os"

	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/redis_client"
	"github.com/daqing/airway/lib/repo/pg"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	appConfig := utils.AppConfig()

	if appConfig.Env == "" {
		log.Println("AIRWAY_ENV is not set")
		os.Exit(1)
	}

	if appConfig.IsLocal {
		envFile := ".env"
		err := godotenv.Load(envFile)
		if err != nil {
			log.Printf("Loading env file: %s failed", envFile)
			os.Exit(2)
		}
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	dsn, err := utils.GetEnv("AIRWAY_PG")
	if err == nil {
		pg.Setup(dsn)
	}

	redisURL, err := utils.GetEnv("AIRWAY_REDIS")
	if err == nil {
		redis_client.Setup(redisURL)
	}

	if len(os.Args) > 1 {
		cmd.Run(os.Args[1:])
	} else {
		runApp()
	}
}

func runApp() {
	app := NewApp("Airway", utils.GetEnvMust("AIRWAY_PORT"))
	app.Run()
}
