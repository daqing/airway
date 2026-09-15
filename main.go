package main

import (
	"errors"
	"log"
	"os"

	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/engine"
	"github.com/daqing/airway/lib/redis_client"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/storage"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// The airway binary is a CLI first: `airway <command>` (see `airway help`).
// Only the `server` subcommand starts the HTTP server.
func main() {
	args := os.Args[1:]

	// `--version` / `-v` print the VERSION file contents directly, without
	// loading .env or any other project setup.
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v") {
		printVersion()
		return
	}

	if len(args) > 0 && args[0] == "server" {
		runServer()
		return
	}

	cmd.Version = versionString()
	loadCLIEnv()
	cmd.Run(args)
}

func runServer() {
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

	dsn := utils.GetEnvMulti("AIRWAY_DSN", "DSN")

	if len(dsn) > 0 {
		if _, setupErr := repo.SetupDB(dsn); setupErr != nil {
			log.Printf("database setup failed: %v", setupErr)
			os.Exit(3)
		}
	}

	redisURL := utils.GetEnvMulti("AIRWAY_REDIS", "REDIS")
	if len(redisURL) > 0 {
		redis_client.Setup(redisURL)
	}

	if _, err := storage.Setup(storage.FromEnv()); err != nil {
		log.Printf("storage setup failed: %v", err)
		os.Exit(4)
	}

	if err := engine.BootAll(); err != nil {
		log.Printf("engine boot failed: %v", err)
		os.Exit(5)
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
