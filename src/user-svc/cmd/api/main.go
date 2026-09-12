package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/haidodev/user-service/internal/authz"
	"github.com/haidodev/user-service/internal/database"
	"github.com/haidodev/user-service/internal/repository"
	"github.com/haidodev/user-service/internal/service"
	"github.com/haidodev/user-service/internal/telemetry"
	userHttp "github.com/haidodev/user-service/internal/transport/http"
)

func main() {
	ctx := context.Background()

	db, err := database.NewPostgresConfig(
		database.Config{
			Host: os.Getenv("DB_HOST"),
			Port: func() int {
				if port := os.Getenv("DB_PORT"); port != "" {
					if p, err := strconv.Atoi(port); err == nil {
						return p
					}
				}
				return 5432
			}(),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	handler := userHttp.NewHandler(userService, authz.NewClient(os.Getenv("AUTHZ_SERVICE_URL")))
	router := gin.New()
	router.Use(telemetry.HTTPMetrics("user-service"))
	telemetry.RegisterMetricsRoute(router)

	// add health probe
	userHttp.RegisterHealthRoutes(router, db)

	// wiredup generated router -> user-service handler
	userHttp.RegisterRoutes(router, handler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Printf("Starting server listening on %d", 8080)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(
		stop,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-stop
	log.Println("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
