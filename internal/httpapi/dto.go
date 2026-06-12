package httpapi

type ErrorResponse struct {
	Error   bool   `json:"error" example:"true"`
	Message string `json:"message" example:"validation failed"`
}

type HealthResponse struct {
	OK      bool   `json:"ok" example:"true"`
	Service string `json:"service" example:"phylax"`
}
