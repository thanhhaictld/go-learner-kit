package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterHealthRoutes(router gin.IRouter, db *gorm.DB) {
	router.GET(
		"/health/live",
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	router.GET(
		"/health/ready",
		func(c *gin.Context) {
			sqlDB, err := db.DB()
			if err != nil {
				c.Status(http.StatusServiceUnavailable)
				return
			}

			ctx, cancel := context.WithTimeout(
				c.Request.Context(),
				time.Second,
			)

			defer cancel()

			if err := sqlDB.PingContext(ctx); err != nil {
				c.Status(http.StatusServiceUnavailable)
				return
			}

			c.Status(http.StatusOK)
		},
	)
}
