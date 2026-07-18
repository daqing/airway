package main

import (
	"fmt"
	"os"
	"strings"
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
	allowedOrigins := []string{"http://127.0.0.1:5173", "http://localhost:5173"}
	if configured := strings.TrimSpace(os.Getenv("AIRWAY_CORS_ORIGINS")); configured != "" {
		allowedOrigins = strings.Split(configured, ",")
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: true,
	})
}
