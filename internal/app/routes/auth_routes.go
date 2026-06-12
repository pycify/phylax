package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pycify/phylax/internal/app/controllers"
	"github.com/pycify/phylax/internal/app/middleware"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
)

func RegisterAuthRoutes(router chi.Router, controller *controllers.AuthController, authService *identityservice.AuthService) {
	router.Route("/v1/auth", func(router chi.Router) {
		router.Post("/accounts", controller.CreateAccount)
		router.Post("/verify-email", controller.VerifyEmail)
		router.Post("/login", controller.Login)
		router.With(middleware.Auth(authService)).Post("/logout", controller.Logout)
	})
}
