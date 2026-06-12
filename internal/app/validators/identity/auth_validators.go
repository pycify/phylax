package identity

type CreateAccountRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	FullName string `json:"full_name" validate:"max=120"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}
