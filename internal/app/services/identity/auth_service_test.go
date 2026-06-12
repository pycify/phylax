package identity

import (
	"context"
	"testing"
	"time"

	identityvalidators "github.com/pycify/phylax/internal/app/validators/identity"
	"github.com/stretchr/testify/require"
)

type captureMailer struct {
	to        string
	code      string
	expiresAt time.Time
}

func (m *captureMailer) SendVerificationOTP(_ context.Context, to string, code string, expiresAt time.Time) error {
	m.to = to
	m.code = code
	m.expiresAt = expiresAt
	return nil
}

func TestAuthServiceCreateVerifyLoginAndLogout(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryAuthRepository()
	otpStore := NewMemoryOTPStore()
	mailer := &captureMailer{}
	service := NewAuthService(repository, otpStore, mailer)

	created, err := service.CreateAccount(ctx, identityvalidators.CreateAccountRequest{
		Email:    " Customer@Phylax.Local ",
		Password: "password",
		FullName: "Development Customer",
	})
	require.NoError(t, err)
	require.Equal(t, "customer@phylax.local", created.User.Email)
	require.Equal(t, "customer@phylax.local", mailer.to)
	require.Len(t, mailer.code, 6)
	require.True(t, mailer.expiresAt.After(time.Now().UTC()))

	_, err = service.Login(ctx, identityvalidators.LoginRequest{
		Email:    "customer@phylax.local",
		Password: "password",
	})
	require.ErrorIs(t, err, ErrEmailNotVerified)

	verified, err := service.VerifyEmail(ctx, identityvalidators.VerifyEmailRequest{
		Email: "customer@phylax.local",
		Code:  mailer.code,
	})
	require.NoError(t, err)
	require.NotEmpty(t, verified.Token)
	require.NotNil(t, verified.User.EmailVerifiedAt)
	foundIdentity := false
	for _, identity := range repository.identities {
		if identity.ProviderUserID == "customer@phylax.local" {
			foundIdentity = true
			require.True(t, identity.EmailVerified)
		}
	}
	require.True(t, foundIdentity)

	user, token, err := service.AuthenticateBearer(ctx, verified.Token)
	require.NoError(t, err)
	require.Equal(t, verified.User.ID, user.ID)
	require.NotEmpty(t, token.ID)

	login, err := service.Login(ctx, identityvalidators.LoginRequest{
		Email:    "customer@phylax.local",
		Password: "password",
	})
	require.NoError(t, err)
	require.NotEmpty(t, login.Token)

	require.NoError(t, service.Logout(ctx, token.ID))
	_, _, err = service.AuthenticateBearer(ctx, verified.Token)
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthServiceRejectsDuplicateEmailAndBadOTP(t *testing.T) {
	ctx := context.Background()
	service := NewAuthService(NewMemoryAuthRepository(), NewMemoryOTPStore(), &captureMailer{})
	payload := identityvalidators.CreateAccountRequest{
		Email:    "customer@phylax.local",
		Password: "password",
	}

	_, err := service.CreateAccount(ctx, payload)
	require.NoError(t, err)
	_, err = service.CreateAccount(ctx, payload)
	require.ErrorIs(t, err, ErrEmailTaken)

	_, err = service.VerifyEmail(ctx, identityvalidators.VerifyEmailRequest{
		Email: "customer@phylax.local",
		Code:  "000000",
	})
	require.ErrorIs(t, err, ErrOTPNotFound)
}
