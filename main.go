package main

import (
	"errors"
	"log"
	"os"

	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/redis_client"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	args := os.Args[1:]
	isCLICommand := len(args) > 0 && args[0] == "cli"

	if isCLICommand {
		loadCLIEnv()
	}

	appConfig := utils.AppConfig()

	if !isCLICommand && appConfig.Env == "" {
		log.Println("AIRWAY_ENV is not set")
		os.Exit(1)
	}

	if !isCLICommand && appConfig.IsLocal {
		envFile := ".env"
		err := godotenv.Load(envFile)
		if err != nil {
			log.Printf("Loading env file: %s failed", envFile)
			os.Exit(2)
		}
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	dsn, err := utils.GetEnv("AIRWAY_DB_DSN")

	if err == nil {
		if _, setupErr := repo.SetupDB(dsn); setupErr != nil {
			log.Printf("database setup failed: %v", setupErr)
			os.Exit(3)
		}
	}

	redisURL, err := utils.GetEnv("AIRWAY_REDIS")
	if err == nil {
		redis_client.Setup(redisURL)
	}

	if len(args) > 0 {
		cmd.Run(args)
	} else {
		runApp()
	}
}

func loadCLIEnv() {
	err := godotenv.Load(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Loading env file: .env failed: %v", err)
	}
}

func runApp() {
	app := NewApp("Airway", utils.GetEnvMust("AIRWAY_PORT"))
	app.Run()
}
