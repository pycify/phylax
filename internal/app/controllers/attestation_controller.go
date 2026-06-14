package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/pycify/phylax/internal/app/helper"
	"github.com/pycify/phylax/internal/app/middleware"
	attestationservice "github.com/pycify/phylax/internal/app/services/attestation"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
)

type AttestationController struct {
	service  *attestationservice.Service
	projects *projectservice.Service
	validate *validator.Validate
}

func NewAttestationController(service *attestationservice.Service) *AttestationController {
	return &AttestationController{service: service, validate: validator.New()}
}

func NewAttestationControllerWithProjects(service *attestationservice.Service, projects *projectservice.Service) *AttestationController {
	return &AttestationController{service: service, projects: projects, validate: validator.New()}
}

// CreateChallenge godoc
//
//	@Summary	Create attestation challenge
//	@Tags		Attestation
//	@Accept		json
//	@Produce	json
//	@Param		app_id	path		string							true	"App id"
//	@Param		payload	body		attestation.CreateChallengeRequest	true	"Challenge payload"
//	@Success	201		{object}	response.CreateChallengeResponse
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	422		{object}	response.ErrorResponse
//	@Router		/v1/apps/{app_id}/challenges [post]
func (c *AttestationController) CreateChallenge(w http.ResponseWriter, req *http.Request) {
	var payload attestationservice.CreateChallengeRequest
	if !helper.DecodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	payload.AppID = chi.URLParam(req, "app_id")

	response, err := c.service.CreateChallenge(req.Context(), payload)
	if err != nil {
		helper.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	helper.WriteJSON(w, http.StatusCreated, response)
}

// VerifyAttestation godoc
//
//	@Summary	Verify provider proof and issue a Phylax token
//	@Tags		Attestation
//	@Accept		json
//	@Produce	json
//	@Param		app_id	path		string							true	"App id"
//	@Param		payload	body		attestation.VerifyAttestationRequest	true	"Attestation payload"
//	@Success	200		{object}	response.VerifyAttestationResponse
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	422		{object}	response.ErrorResponse
//	@Router		/v1/apps/{app_id}/attestations/verify [post]
func (c *AttestationController) VerifyAttestation(w http.ResponseWriter, req *http.Request) {
	var payload attestationservice.VerifyAttestationRequest
	if !helper.DecodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	payload.AppID = chi.URLParam(req, "app_id")

	response, err := c.service.VerifyAttestation(req.Context(), payload)
	if err != nil {
		helper.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	helper.WriteJSON(w, http.StatusOK, response)
}

// Introspect godoc
//
//	@Summary	Verify a Phylax session token
//	@Tags		Sessions
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		payload	body		attestation.IntrospectRequest	true	"Token payload"
//	@Success	200		{object}	response.IntrospectResponse
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	422		{object}	response.ErrorResponse
//	@Router		/v1/sessions/introspect [post]
func (c *AttestationController) Introspect(w http.ResponseWriter, req *http.Request) {
	var payload attestationservice.IntrospectRequest
	if !helper.DecodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	result := c.service.Introspect(req.Context(), payload.Token)
	if result.Active && c.projects != nil {
		apiKey, ok := middleware.CurrentProjectAPIKey(req)
		if !ok || apiKey == nil {
			helper.WriteError(w, http.StatusUnauthorized, "Project API key is required")
			return
		}
		if result.Session == nil || c.projects.ProjectOwnsApp(req.Context(), apiKey.ProjectID, result.Session.AppID) != nil {
			helper.WriteError(w, http.StatusForbidden, "Phylax token does not belong to this project")
			return
		}
	}
	helper.WriteJSON(w, http.StatusOK, result)
}
