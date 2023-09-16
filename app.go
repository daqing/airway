package main

import (
	"os"

	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

type App struct {
	r *gin.Engine
}

func NewApp() *App {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	router := gin.New()

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
