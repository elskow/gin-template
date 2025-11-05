package account

import (
	"github.com/elskow/go-microservice-template/middlewares"
	"github.com/elskow/go-microservice-template/modules/account/controller"
	"github.com/elskow/go-microservice-template/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
)

func RegisterRoutes(server gin.IRouter, injector *do.Injector) {
	ctrl := do.MustInvokeNamed[*controller.Controller](injector, "controller")
	jwtService := do.MustInvokeNamed[jwt.Service](injector, "jwt-service")

	public := server.Group("/account")
	{
		public.POST("/register", ctrl.Register)
		public.POST("/login", ctrl.Login)
		public.POST("/refresh", ctrl.RefreshToken)
	}

	protected := server.Group("/account")
	protected.Use(middlewares.Authenticate(jwtService))
	{
		protected.POST("/logout", ctrl.Logout)
		protected.GET("/me", ctrl.Me)
		protected.PUT("/me", ctrl.UpdateUser)
		protected.DELETE("/me", ctrl.DeleteUser)
	}
}
