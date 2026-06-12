package controllers

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/pycify/phylax/internal/app/middleware"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
	identitydto "github.com/pycify/phylax/internal/app/types/dto/identity"
	identityvalidators "github.com/pycify/phylax/internal/app/validators/identity"
)

type AuthController struct {
	service  *identityservice.AuthService
	validate *validator.Validate
}

func NewAuthController(service *identityservice.AuthService) *AuthController {
	return &AuthController{service: service, validate: validator.New()}
}

// CreateAccount godoc
//
//	@Summary	Create standard account
//	@Tags		Auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		identity.CreateAccountRequest	true	"Account payload"
//	@Success	201		{object}	identity.AccountCreatedResponse
//	@Failure	422		{object}	httpapi.ErrorResponse
//	@Router		/v1/auth/accounts [post]
func (c *AuthController) CreateAccount(w http.ResponseWriter, req *http.Request) {
	var payload identityvalidators.CreateAccountRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	result, err := c.service.CreateAccount(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, identitydto.AccountCreatedResponse{
		User:      identitydto.NewUserDTO(result.User),
		ExpiresAt: result.ExpiresAt,
	})
}

// VerifyEmail godoc
//
//	@Summary	Verify standard account email
//	@Tags		Auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		identity.VerifyEmailRequest	true	"OTP payload"
//	@Success	200		{object}	identity.AuthTokenResponse
//	@Failure	422		{object}	httpapi.ErrorResponse
//	@Router		/v1/auth/verify-email [post]
func (c *AuthController) VerifyEmail(w http.ResponseWriter, req *http.Request) {
	var payload identityvalidators.VerifyEmailRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	result, err := c.service.VerifyEmail(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, identitydto.AuthTokenResponse{
		Token: result.Token,
		User:  identitydto.NewUserDTO(result.User),
	})
}

// Login godoc
//
//	@Summary	Login with standard provider
//	@Tags		Auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		identity.LoginRequest	true	"Login payload"
//	@Success	200		{object}	identity.AuthTokenResponse
//	@Failure	401		{object}	httpapi.ErrorResponse
//	@Router		/v1/auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, req *http.Request) {
	var payload identityvalidators.LoginRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	result, err := c.service.Login(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, identitydto.AuthTokenResponse{
		Token: result.Token,
		User:  identitydto.NewUserDTO(result.User),
	})
}

// Logout godoc
//
//	@Summary	Logout current access token
//	@Tags		Auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	204
//	@Failure	401	{object}	httpapi.ErrorResponse
//	@Router		/v1/auth/logout [post]
func (c *AuthController) Logout(w http.ResponseWriter, req *http.Request) {
	token, ok := middleware.CurrentAccessToken(req)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Authenticated token missing from context")
		return
	}
	if err := c.service.Logout(req.Context(), token.ID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
