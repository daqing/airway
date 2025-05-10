package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/daqing/airway/config"
	"github.com/daqing/airway/lib/repo/pg"
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
		envFile := ".env"
		err := godotenv.Load(envFile)
		if err != nil {
			log.Printf("Loading env file: %s failed", envFile)
			os.Exit(2)
		}
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	dsn, err := utils.GetEnv("AIRWAY_DSN")
	if err == nil {
		pg.Setup(dsn)
	}

	port := utils.GetEnvMust("AIRWAY_PORT")
	fmt.Printf("Airway running at: http://127.0.0.1:%s\n", port)

	app := NewApp()
	app.Run(":" + port)
}

type App struct {
	r *gin.Engine
}

func NewApp() *App {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORS())

	config.Routes(router)

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

// Default CORS middleware
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: true,
	})
}
