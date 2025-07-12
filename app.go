package main

import (
	"fmt"
	"time"

	"github.com/daqing/airway/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type App struct {
	r    *gin.Engine
	name string // Application name
	port string
}

func NewApp(name, port string) *App {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORS())

	config.Routes(router)

	return &App{
		r:    router,
		name: name,
		port: port,
	}
}

func (a *App) Router() *gin.Engine {
	return a.r
}

func (a *App) Run() {
	fmt.Printf("%s running at: http://127.0.0.1:%s\n", a.name, a.port)
	a.r.Run(":" + a.port)
}

// Default CORS middleware
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: true,
	})
}
