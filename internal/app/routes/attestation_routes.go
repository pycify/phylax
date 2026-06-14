package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pycify/phylax/internal/app/controllers"
	"github.com/pycify/phylax/internal/app/middleware"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
)

func RegisterAttestationRoutes(router chi.Router, controller *controllers.AttestationController, projectService *projectservice.Service) {
	router.With(middleware.AttestationRateLimit()).Post("/v1/apps/{app_id}/challenges", controller.CreateChallenge)
	router.Post("/v1/apps/{app_id}/attestations/verify", controller.VerifyAttestation)
	router.With(middleware.AttestationRateLimit()).Post("/v1/apps/{app_id}/attestations/refresh", controller.RefreshAttestation)
	router.With(middleware.ProjectAPIKey(projectService)).Post("/v1/sessions/introspect", controller.Introspect)
}
