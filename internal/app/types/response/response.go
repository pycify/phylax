package response

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
