package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/pycify/phylax/docs"
	"github.com/pycify/phylax/internal/app/controllers"
	approutes "github.com/pycify/phylax/internal/app/routes"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
	"github.com/pycify/phylax/internal/app/types/response"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Router struct {
	router                chi.Router
	authService           *identityservice.AuthService
	authController        *controllers.AuthController
	projects              *controllers.ProjectController
	attestationController *controllers.AttestationController
}

func NewRouter(
	attestationController *controllers.AttestationController,
	authService *identityservice.AuthService,
	authController *controllers.AuthController,
	projectController *controllers.ProjectController,
) http.Handler {
	router := &Router{
		router:                chi.NewRouter(),
		authService:           authService,
		authController:        authController,
		projects:              projectController,
		attestationController: attestationController,
	}
	router.routes()
	return router.router
}

func (r *Router) routes() {
	r.router.Use(chimiddleware.RequestID)
	r.router.Use(chimiddleware.RealIP)
	r.router.Use(chimiddleware.Recoverer)

	r.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.PersistAuthorization(true),
	))
	r.router.Get("/v1/health", r.health)
	r.router.Get("/v1/openapi.json", r.openapi)
	if r.attestationController != nil {
		approutes.RegisterAttestationRoutes(r.router, r.attestationController)
	}
	if r.authController != nil && r.authService != nil {
		approutes.RegisterAuthRoutes(r.router, r.authController, r.authService)
	}
	if r.projects != nil && r.authService != nil {
		approutes.RegisterProjectRoutes(r.router, r.projects, r.authService)
	}
}

func (r *Router) openapi(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "/swagger/doc.json", http.StatusTemporaryRedirect)
}

// health godoc
//
//	@Summary	Service health
//	@Tags		System
//	@Produce	json
//	@Success	200	{object}	response.APIResponse[httpapi.HealthResponse]
//	@Router		/v1/health [get]
func (r *Router) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		OK:      true,
		Service: "phylax",
	})
}

func writeJSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response.APIResponse[T]{
		Error:   false,
		Message: "success",
		Data:    &data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response.ErrorResponse{
		Error:   true,
		Message: message,
	})
}
