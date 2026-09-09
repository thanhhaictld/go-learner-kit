package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/haidodev/authz-service/internal/authz"
	transport "github.com/haidodev/authz-service/internal/transport/http"
)

func main() {
	bootstrapOrg, err := uuid.Parse(os.Getenv("BOOTSTRAP_ORGANIZATION_ID"))
	if err != nil {
		log.Fatal("BOOTSTRAP_ORGANIZATION_ID must be a UUID")
	}
	bootstrapUser, err := uuid.Parse(os.Getenv("BOOTSTRAP_ADMIN_USER_ID"))
	if err != nil {
		log.Fatal("BOOTSTRAP_ADMIN_USER_ID must be a UUID")
	}

	engine := authz.NewOpenFGA(authz.Config{
		APIURL:                  os.Getenv("FGA_API_URL"),
		StoreName:               os.Getenv("FGA_STORE_NAME"),
		BootstrapOrganizationID: bootstrapOrg,
		BootstrapAdminUserID:    bootstrapUser,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := initializeEngine(ctx, engine); err != nil {
		log.Fatal(err)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	transport.NewHandler(engine, bootstrapOrg, bootstrapUser).Register(router)
	server := &http.Server{Addr: ":8080", Handler: router}
	go func() {
		log.Printf("authorization service listening on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}

func initializeEngine(ctx context.Context, engine authz.Engine) error {
	var lastErr error
	for {
		if err := engine.Initialize(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(500 * time.Millisecond):
		}
	}
}
