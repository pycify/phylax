package identity

import (
	"time"

	identitymodels "github.com/pycify/phylax/internal/app/models/identity"
	"github.com/pycify/phylax/internal/app/types/enum"
)

type UserDTO struct {
	ID              string          `json:"id"`
	Email           string          `json:"email"`
	FullName        string          `json:"full_name"`
	UserType        enum.UserType   `json:"user_type"`
	Status          enum.UserStatus `json:"status"`
	EmailVerifiedAt *time.Time      `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func NewUserDTO(user identitymodels.User) UserDTO {
	return UserDTO{
		ID:              user.ID,
		Email:           user.Email,
		FullName:        user.FullName,
		UserType:        user.UserType,
		Status:          user.Status,
		EmailVerifiedAt: user.EmailVerifiedAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

type AccountCreatedResponse struct {
	User      UserDTO   `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthTokenResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}
