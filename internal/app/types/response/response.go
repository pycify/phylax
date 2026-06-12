package response

import (
	identitydto "github.com/pycify/phylax/internal/app/types/dto/identity"
	projectdto "github.com/pycify/phylax/internal/app/types/dto/projects"
	"github.com/pycify/phylax/internal/phylax"
)

type APIResponse[T any] struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    *T     `json:"data"`
}

type ErrorResponse struct {
	Error   bool   `json:"error" example:"true"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type HealthData struct {
	OK      bool   `json:"ok" example:"true"`
	Service string `json:"service" example:"phylax"`
}

type HealthResponse struct {
	Error   bool       `json:"error" example:"false"`
	Message string     `json:"message" example:"success"`
	Data    HealthData `json:"data"`
}

type AccountCreatedResponse struct {
	Error   bool                               `json:"error" example:"false"`
	Message string                             `json:"message" example:"success"`
	Data    identitydto.AccountCreatedResponse `json:"data"`
}

type AuthTokenResponse struct {
	Error   bool                          `json:"error" example:"false"`
	Message string                        `json:"message" example:"success"`
	Data    identitydto.AuthTokenResponse `json:"data"`
}

type ProjectResponse struct {
	Error   bool                       `json:"error" example:"false"`
	Message string                     `json:"message" example:"success"`
	Data    projectdto.ProjectResponse `json:"data"`
}

type ProjectsResponse struct {
	Error   bool                        `json:"error" example:"false"`
	Message string                      `json:"message" example:"success"`
	Data    projectdto.ProjectsResponse `json:"data"`
}

type ProjectAppResponse struct {
	Error   bool                          `json:"error" example:"false"`
	Message string                        `json:"message" example:"success"`
	Data    projectdto.ProjectAppResponse `json:"data"`
}

type ProjectAppsResponse struct {
	Error   bool                           `json:"error" example:"false"`
	Message string                         `json:"message" example:"success"`
	Data    projectdto.ProjectAppsResponse `json:"data"`
}

type AppAttestationStatusResponse struct {
	Error   bool                                    `json:"error" example:"false"`
	Message string                                  `json:"message" example:"success"`
	Data    projectdto.AppAttestationStatusResponse `json:"data"`
}

type CreateChallengeResponse struct {
	Error   bool                           `json:"error" example:"false"`
	Message string                         `json:"message" example:"success"`
	Data    phylax.CreateChallengeResponse `json:"data"`
}

type VerifyAttestationResponse struct {
	Error   bool                             `json:"error" example:"false"`
	Message string                           `json:"message" example:"success"`
	Data    phylax.VerifyAttestationResponse `json:"data"`
}

type IntrospectResponse struct {
	Error   bool                      `json:"error" example:"false"`
	Message string                    `json:"message" example:"success"`
	Data    phylax.IntrospectResponse `json:"data"`
}
