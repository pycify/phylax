package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pycify/phylax/internal/app/controllers"
)

func RegisterAttestationRoutes(router chi.Router, controller *controllers.AttestationController) {
	router.Post("/v1/challenges", controller.CreateChallenge)
	router.Post("/v1/attestations/verify", controller.VerifyAttestation)
	router.Post("/v1/sessions/introspect", controller.Introspect)
}
