package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/daqing/airway/config"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	sloggin "github.com/samber/slog-gin"
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

type App struct {
	r *gin.Engine
}

func NewApp() *App {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	router := gin.New()

	router.Static("/public", "./public")

	router.Use(sloggin.New(logger))
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	return &App{
		r: router,
	}
}

func (a *App) Router() *gin.Engine {
	return a.r
}

func (a *App) Run(addr string) {
	a.r.Run(addr)
}
