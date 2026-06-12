package httpapi

type HealthResponse struct {
	OK      bool   `json:"ok" example:"true"`
	Service string `json:"service" example:"phylax"`
}
