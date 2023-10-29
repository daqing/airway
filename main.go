package main

import (
	"fmt"
	"log"
	"os"

	"github.com/daqing/airway/config"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-contrib/cors"
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
		envFile := fmt.Sprintf(".env.%s", appConfig.Env)
		err := godotenv.Load(envFile)
		if err != nil {
			log.Printf("Loading env file: %s failed", envFile)
			os.Exit(2)
		}
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	repo.Setup()

	app := NewApp()

	r := app.Router()

	r.Use(cors.Default())

	config.Routes(r)

	var port = os.Getenv("AIRWAY_PORT")

	if appConfig.IsLocal {
		fmt.Printf("Airway running at: http://127.0.0.1:%s\n", port)
	}

	app.Run(":" + port)
}
