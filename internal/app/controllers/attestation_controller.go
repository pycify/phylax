package controllers

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	attestationservice "github.com/pycify/phylax/internal/app/services/attestation"
	"github.com/pycify/phylax/internal/phylax"
)

type AttestationController struct {
	service  *attestationservice.Service
	validate *validator.Validate
}

func NewAttestationController(service *attestationservice.Service) *AttestationController {
	return &AttestationController{service: service, validate: validator.New()}
}

// CreateChallenge godoc
//
//	@Summary	Create attestation challenge
//	@Tags		Attestation
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		phylax.CreateChallengeRequest	true	"Challenge payload"
//	@Success	201		{object}	response.APIResponse[phylax.CreateChallengeResponse]
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	422		{object}	response.ErrorResponse
//	@Router		/v1/challenges [post]
func (c *AttestationController) CreateChallenge(w http.ResponseWriter, req *http.Request) {
	var payload phylax.CreateChallengeRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}

	response, err := c.service.CreateChallenge(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

// VerifyAttestation godoc
//
//	@Summary	Verify provider proof and issue a Phylax token
//	@Tags		Attestation
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		phylax.VerifyAttestationRequest	true	"Attestation payload"
//	@Success	200		{object}	response.APIResponse[phylax.VerifyAttestationResponse]
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	422		{object}	response.ErrorResponse
//	@Router		/v1/attestations/verify [post]
func (c *AttestationController) VerifyAttestation(w http.ResponseWriter, req *http.Request) {
	var payload phylax.VerifyAttestationRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}

	response, err := c.service.VerifyAttestation(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// Introspect godoc
//
//	@Summary	Verify a Phylax session token
//	@Tags		Sessions
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		phylax.IntrospectRequest	true	"Token payload"
//	@Success	200		{object}	response.APIResponse[phylax.IntrospectResponse]
//	@Failure	400		{object}	response.ErrorResponse
//	@Failure	422		{object}	response.ErrorResponse
//	@Router		/v1/sessions/introspect [post]
func (c *AttestationController) Introspect(w http.ResponseWriter, req *http.Request) {
	var payload phylax.IntrospectRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	writeJSON(w, http.StatusOK, c.service.Introspect(req.Context(), payload.Token))
}
