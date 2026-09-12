package http

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/haidodev/user-service/api/generated"
)

func RegisterRoutes(router gin.IRouter, handler *Handler) {
	generated.RegisterHandlersWithOptions(router, handler, generated.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			if strings.Contains(strings.ToLower(err.Error()), "header parameter") {
				writeIdentityError(c, errMissingIdentityHeader)
				return
			}
			c.JSON(statusCode, generated.Error{Code: "invalid_request", Message: err.Error()})
		},
	})
}
