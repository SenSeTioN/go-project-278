package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	if os.Getenv("SENTRY_DSN") != "" {
		router.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	}

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	router.GET("/debug-sentry", func(c *gin.Context) {
		panic("sentry test panic")
	})

	return router
}

func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return
	}
	if err := sentry.Init(sentry.ClientOptions{Dsn: dsn}); err != nil {
		log.Printf("sentry.Init: %v", err)
	}
}

func main() {
	initSentry()
	defer sentry.Flush(2 * time.Second)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := setupRouter().Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
