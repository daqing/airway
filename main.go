package main

import (
	"fmt"
	"log"
	"os"

	"github.com/daqing/airway/config"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	env := os.Getenv("AIRWAY_ENV")
	if env == "" {
		log.Println("AIRWAY_ENV was not set")
		os.Exit(1)
	}

	if env == "local" {
		envFile := fmt.Sprintf(".env.%s", env)
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

	app.Run(port())
}

func port() string {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatalf("No PORT env set")
	}

	return ":" + port
}
