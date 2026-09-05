package main

import (
	"errors"
	"log"
	"os"

	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/redis_client"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/storage"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	args := os.Args[1:]
	isCLICommand := len(args) > 0 && args[0] == "cli"

	if isCLICommand {
		loadCLIEnv()
		cmd.Run(args)
		return
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

	dsn := utils.GetEnvOr("AIRWAY_DSN", "DSN")

	if len(dsn) > 0 {
		if _, setupErr := repo.SetupDB(dsn); setupErr != nil {
			log.Printf("database setup failed: %v", setupErr)
			os.Exit(3)
		}
	}

	redisURL := utils.GetEnvOr("AIRWAY_REDIS", "REDIS")
	if len(redisURL) > 0 {
		redis_client.Setup(redisURL)
	}

	if _, err := storage.Setup(storage.FromEnv()); err != nil {
		log.Printf("storage setup failed: %v", err)
		os.Exit(4)
	}

	if len(args) > 0 {
		cmd.Run(args)
		return
	}

	runApp()
}

func loadCLIEnv() {
	err := godotenv.Load(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Loading env file: .env failed: %v", err)
	}
}

func runApp() {
	app := NewApp("Airway", utils.GetEnvOr("AIRWAY_PORT", "PORT"))
	app.Run()
}
