package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	_ "github.com/pycify/phylax/docs"
	"github.com/pycify/phylax/internal/app/controllers"
	approutes "github.com/pycify/phylax/internal/app/routes"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
	"github.com/pycify/phylax/internal/phylax"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Router struct {
	service        *phylax.Service
	router         chi.Router
	validate       *validator.Validate
	authService    *identityservice.AuthService
	authController *controllers.AuthController
	projects       *controllers.ProjectController
}

func NewRouter(
	service *phylax.Service,
	authService *identityservice.AuthService,
	authController *controllers.AuthController,
	projectController *controllers.ProjectController,
) http.Handler {
	router := &Router{
		service:        service,
		router:         chi.NewRouter(),
		validate:       validator.New(),
		authService:    authService,
		authController: authController,
		projects:       projectController,
	}
	router.routes()
	return router.router
}

func (r *Router) routes() {
	r.router.Use(chimiddleware.RequestID)
	r.router.Use(chimiddleware.RealIP)
	r.router.Use(chimiddleware.Recoverer)

	r.router.Get("/swagger/*", httpSwagger.WrapHandler)
	r.router.Get("/v1/health", r.health)
	r.router.Get("/v1/openapi.json", r.openapi)
	r.router.Post("/v1/challenges", r.createChallenge)
	r.router.Post("/v1/attestations/verify", r.verifyAttestation)
	r.router.Post("/v1/sessions/introspect", r.introspect)
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
//	@Success	200	{object}	HealthResponse
//	@Router		/v1/health [get]
func (r *Router) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		OK:      true,
		Service: "phylax",
	})
}

// createChallenge godoc
//
//	@Summary	Create attestation challenge
//	@Tags		Attestation
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		phylax.CreateChallengeRequest	true	"Challenge payload"
//	@Success	201		{object}	phylax.CreateChallengeResponse
//	@Failure	400		{object}	ErrorResponse
//	@Failure	422		{object}	ErrorResponse
//	@Router		/v1/challenges [post]
func (r *Router) createChallenge(w http.ResponseWriter, req *http.Request) {
	var payload phylax.CreateChallengeRequest
	if err := readJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := r.validate.Struct(payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	response, err := r.service.CreateChallenge(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

// verifyAttestation godoc
//
//	@Summary	Verify provider proof and issue a Phylax token
//	@Tags		Attestation
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		phylax.VerifyAttestationRequest	true	"Attestation payload"
//	@Success	200		{object}	phylax.VerifyAttestationResponse
//	@Failure	400		{object}	ErrorResponse
//	@Failure	422		{object}	ErrorResponse
//	@Router		/v1/attestations/verify [post]
func (r *Router) verifyAttestation(w http.ResponseWriter, req *http.Request) {
	var payload phylax.VerifyAttestationRequest
	if err := readJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := r.validate.Struct(payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	response, err := r.service.VerifyAttestation(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// introspect godoc
//
//	@Summary	Verify a Phylax session token
//	@Tags		Sessions
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		phylax.IntrospectRequest	true	"Token payload"
//	@Success	200		{object}	phylax.IntrospectResponse
//	@Failure	400		{object}	ErrorResponse
//	@Failure	422		{object}	ErrorResponse
//	@Router		/v1/sessions/introspect [post]
func (r *Router) introspect(w http.ResponseWriter, req *http.Request) {
	var payload phylax.IntrospectRequest
	if err := readJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := r.validate.Struct(payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, r.service.Introspect(req.Context(), payload.Token))
}

func readJSON(req *http.Request, out any) error {
	defer req.Body.Close()
	return json.NewDecoder(req.Body).Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":   true,
		"message": message,
	})
}
