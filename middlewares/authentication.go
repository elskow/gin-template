package middlewares

import (
	"net/http"
	"strings"

	"github.com/elskow/go-microservice-template/pkg/constants"
	"github.com/elskow/go-microservice-template/pkg/jwt"
	"github.com/elskow/go-microservice-template/pkg/response"
	"github.com/gin-gonic/gin"
)

func Authenticate(jwtService jwt.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")

		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Error[any](
				response.ErrCodeUnauthorized,
				"token not found",
			))
			return
		}

		if !strings.Contains(authHeader, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Error[any](
				response.ErrCodeUnauthorized,
				"invalid token format",
			))
			return
		}

		authHeader = strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwtService.ValidateToken(authHeader)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Error[any](
				response.ErrCodeUnauthorized,
				"invalid token",
			))
			return
		}

		if !token.Valid {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Error[any](
				response.ErrCodeUnauthorized,
				"access denied",
			))
			return
		}

		userID, err := jwtService.GetUserIDByToken(authHeader)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, response.Error[any](
				response.ErrCodeUnauthorized,
				err.Error(),
			))
			return
		}

		ctx.Set(constants.CtxKeyToken, authHeader)
		ctx.Set(constants.CtxKeyUserID, userID)
		ctx.Next()
	}
}
