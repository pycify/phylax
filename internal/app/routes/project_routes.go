package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pycify/phylax/internal/app/controllers"
	"github.com/pycify/phylax/internal/app/middleware"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
)

func RegisterProjectRoutes(router chi.Router, controller *controllers.ProjectController, authService *identityservice.AuthService) {
	router.Route("/v1/projects", func(router chi.Router) {
		router.Use(middleware.Auth(authService))
		router.Get("/", controller.ListProjects)
		router.Post("/", controller.CreateProject)
		router.Get("/{project_id}", controller.GetProject)
		router.Patch("/{project_id}", controller.UpdateProject)
		router.Delete("/{project_id}", controller.DeleteProject)

		router.Get("/{project_id}/apps", controller.ListProjectApps)
		router.Post("/{project_id}/apps", controller.CreateProjectApp)
	})

	router.Route("/v1/apps", func(router chi.Router) {
		router.Use(middleware.Auth(authService))
		router.Get("/{app_id}", controller.GetProjectApp)
		router.Patch("/{app_id}", controller.UpdateProjectApp)
		router.Delete("/{app_id}", controller.DeleteProjectApp)
		router.Get("/{app_id}/steps", controller.GetProjectAppAttestationStatus)
	})
}
