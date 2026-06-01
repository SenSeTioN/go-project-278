// Package main — точка входа сервиса URL-сокращателя.
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/SenSeTioN/go-project-278/internal/db"
	"github.com/SenSeTioN/go-project-278/internal/handlers"
)

// corsOrigins возвращает список разрешённых Origin для CORS-middleware.
func corsOrigins(baseURL string) []string {
	origins := []string{"http://localhost:5173"}
	if baseURL != "" && baseURL != "http://localhost:5173" {
		origins = append(origins, baseURL)
	}
	return origins
}

// setupRouter собирает HTTP-роутер Gin со всеми middleware и маршрутами.
func setupRouter(q db.Querier, baseURL string) *gin.Engine {
	router := gin.New()
	router.TrustedPlatform = gin.PlatformCloudflare
	router.Use(gin.Logger(), gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins(baseURL),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Range"},
		ExposeHeaders:    []string{"Content-Range", "Accept-Ranges"},
		AllowCredentials: false,
	}))

	if os.Getenv("SENTRY_DSN") != "" {
		router.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	}

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	router.GET("/debug-sentry", func(c *gin.Context) {
		panic("sentry test panic")
	})

	if q != nil {
		handlers.New(q, baseURL).Register(router)
		handlers.NewVisits(q).Register(router)
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	return router
}

// initSentry инициализирует Sentry SDK, если задана переменная окружения
func initSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return
	}
	if err := sentry.Init(sentry.ClientOptions{Dsn: dsn}); err != nil {
		log.Printf("sentry.Init: %v", err)
	}
}

// openDB открывает пул соединений к PostgreSQL по указанному DSN и проверяет его работоспособность.
func openDB(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func main() {
	_ = godotenv.Load()

	initSentry()
	defer sentry.Flush(2 * time.Second)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	var queries db.Querier
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		conn, err := openDB(dsn)
		if err != nil {
			log.Fatalf("db connect: %v", err)
		}
		defer func() { _ = conn.Close() }()
		queries = db.New(conn)
		log.Println("database connected")
	} else {
		log.Println("DATABASE_URL not set, /api/links disabled")
	}

	if err := setupRouter(queries, baseURL).Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
