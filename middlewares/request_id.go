package middlewares

import (
	"github.com/elskow/go-microservice-template/pkg/constants"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(constants.CtxKeyRequestID, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
