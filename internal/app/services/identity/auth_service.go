package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	identitymodels "github.com/pycify/phylax/internal/app/models/identity"
	"github.com/pycify/phylax/internal/app/types/enum"
	identityvalidators "github.com/pycify/phylax/internal/app/validators/identity"
	"github.com/pycify/phylax/internal/phylax"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const emailVerificationTTL = 5 * time.Minute

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailNotVerified   = errors.New("email has not been verified")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("an account already exists for this email")
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user identitymodels.User) error
	UpdateUser(ctx context.Context, user identitymodels.User) error
	FindUserByEmail(ctx context.Context, email string) (*identitymodels.User, error)
	FindUserByID(ctx context.Context, id string) (*identitymodels.User, error)
	CreateIdentity(ctx context.Context, identity identitymodels.UserIdentity) error
	MarkIdentityEmailVerified(ctx context.Context, provider enum.AuthProvider, providerUserID string) error
	CreateAccessToken(ctx context.Context, token identitymodels.AuthAccessToken) error
	FindAccessTokenByHash(ctx context.Context, hash string) (*identitymodels.AuthAccessToken, error)
	UpdateAccessToken(ctx context.Context, token identitymodels.AuthAccessToken) error
	RevokeAccessToken(ctx context.Context, tokenID string) error
}

type AuthService struct {
	repository AuthRepository
	otpStore   OTPStore
	mailer     Mailer
}

type AccountCreated struct {
	User      identitymodels.User
	ExpiresAt time.Time
}

type AuthResult struct {
	Token string
	User  identitymodels.User
}

func NewAuthService(repository AuthRepository, otpStore OTPStore, mailer Mailer) *AuthService {
	return &AuthService{repository: repository, otpStore: otpStore, mailer: mailer}
}

func (s *AuthService) CreateAccount(ctx context.Context, payload identityvalidators.CreateAccountRequest) (AccountCreated, error) {
	email := normalizeEmail(payload.Email)
	if existing, err := s.repository.FindUserByEmail(ctx, email); err == nil && existing != nil {
		return AccountCreated{}, ErrEmailTaken
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return AccountCreated{}, err
	}
	now := time.Now().UTC()
	user := identitymodels.User{
		ID:           phylax.RandomID("usr"),
		Email:        email,
		FullName:     strings.TrimSpace(payload.FullName),
		PasswordHash: string(passwordHash),
		UserType:     enum.UserTypeCustomer,
		Status:       enum.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repository.CreateUser(ctx, user); err != nil {
		return AccountCreated{}, err
	}
	identity := identitymodels.UserIdentity{
		ID:             phylax.RandomID("idt"),
		UserID:         user.ID,
		Provider:       enum.AuthProviderStandard,
		ProviderUserID: email,
		Email:          email,
		EmailVerified:  false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repository.CreateIdentity(ctx, identity); err != nil {
		return AccountCreated{}, err
	}
	expiresAt, err := s.issueVerificationOTP(ctx, email)
	if err != nil {
		return AccountCreated{}, err
	}
	return AccountCreated{User: user, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, payload identityvalidators.VerifyEmailRequest) (AuthResult, error) {
	email := normalizeEmail(payload.Email)
	if err := s.otpStore.Verify(ctx, "email_verification", email, payload.Code); err != nil {
		return AuthResult{}, err
	}
	user, err := s.repository.FindUserByEmail(ctx, email)
	if err != nil {
		return AuthResult{}, err
	}
	now := time.Now().UTC()
	user.EmailVerifiedAt = &now
	user.UpdatedAt = now
	if err := s.repository.UpdateUser(ctx, *user); err != nil {
		return AuthResult{}, err
	}
	if err := s.repository.MarkIdentityEmailVerified(ctx, enum.AuthProviderStandard, email); err != nil {
		return AuthResult{}, err
	}
	token, err := s.createAccessToken(ctx, user.ID, "standard")
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: token, User: *user}, nil
}

func (s *AuthService) Login(ctx context.Context, payload identityvalidators.LoginRequest) (AuthResult, error) {
	user, err := s.repository.FindUserByEmail(ctx, normalizeEmail(payload.Email))
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	if user.EmailVerifiedAt == nil {
		return AuthResult{}, ErrEmailNotVerified
	}
	token, err := s.createAccessToken(ctx, user.ID, "standard")
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Token: token, User: *user}, nil
}

func (s *AuthService) AuthenticateBearer(ctx context.Context, bearerToken string) (*identitymodels.User, *identitymodels.AuthAccessToken, error) {
	hash := hashToken(bearerToken)
	token, err := s.repository.FindAccessTokenByHash(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	if token.RevokedAt != nil || (token.ExpiresAt != nil && time.Now().UTC().After(*token.ExpiresAt)) {
		return nil, nil, ErrInvalidCredentials
	}
	user, err := s.repository.FindUserByID(ctx, token.UserID)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	token.LastUsedAt = &now
	token.UpdatedAt = now
	_ = s.repository.UpdateAccessToken(ctx, *token)
	return user, token, nil
}

func (s *AuthService) Logout(ctx context.Context, tokenID string) error {
	return s.repository.RevokeAccessToken(ctx, tokenID)
}

func (s *AuthService) issueVerificationOTP(ctx context.Context, email string) (time.Time, error) {
	code := generateOTP()
	expiresAt := time.Now().UTC().Add(emailVerificationTTL)
	if err := s.otpStore.Set(ctx, "email_verification", email, code, emailVerificationTTL); err != nil {
		return time.Time{}, err
	}
	if err := s.mailer.SendVerificationOTP(ctx, email, code, expiresAt); err != nil {
		return time.Time{}, err
	}
	return expiresAt, nil
}

func (s *AuthService) createAccessToken(ctx context.Context, userID string, name string) (string, error) {
	plain := "phylax_" + randomString(36)
	now := time.Now().UTC()
	token := identitymodels.AuthAccessToken{
		ID:        phylax.RandomID("tok"),
		UserID:    userID,
		Name:      name,
		Hash:      hashToken(plain),
		Abilities: "[]",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repository.CreateAccessToken(ctx, token); err != nil {
		return "", err
	}
	return plain, nil
}

type MemoryAuthRepository struct {
	mu         sync.RWMutex
	users      map[string]identitymodels.User
	identities map[string]identitymodels.UserIdentity
	tokens     map[string]identitymodels.AuthAccessToken
}

func NewMemoryAuthRepository() *MemoryAuthRepository {
	return &MemoryAuthRepository{
		users:      map[string]identitymodels.User{},
		identities: map[string]identitymodels.UserIdentity{},
		tokens:     map[string]identitymodels.AuthAccessToken{},
	}
}

func (r *MemoryAuthRepository) CreateUser(_ context.Context, user identitymodels.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}

func (r *MemoryAuthRepository) UpdateUser(_ context.Context, user identitymodels.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}

func (r *MemoryAuthRepository) FindUserByEmail(_ context.Context, email string) (*identitymodels.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Email == normalizeEmail(email) && user.DeletedAt.Time.IsZero() {
			return &user, nil
		}
	}
	return nil, ErrUserNotFound
}

func (r *MemoryAuthRepository) FindUserByID(_ context.Context, id string) (*identitymodels.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok || !user.DeletedAt.Time.IsZero() {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *MemoryAuthRepository) CreateIdentity(_ context.Context, identity identitymodels.UserIdentity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.identities[identity.ID] = identity
	return nil
}

func (r *MemoryAuthRepository) MarkIdentityEmailVerified(_ context.Context, provider enum.AuthProvider, providerUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, identity := range r.identities {
		if identity.Provider == provider && identity.ProviderUserID == providerUserID && identity.DeletedAt.Time.IsZero() {
			identity.EmailVerified = true
			identity.UpdatedAt = time.Now().UTC()
			r.identities[id] = identity
			return nil
		}
	}
	return ErrUserNotFound
}

func (r *MemoryAuthRepository) CreateAccessToken(_ context.Context, token identitymodels.AuthAccessToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token.Hash] = token
	return nil
}

func (r *MemoryAuthRepository) FindAccessTokenByHash(_ context.Context, hash string) (*identitymodels.AuthAccessToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token, ok := r.tokens[hash]
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return &token, nil
}

func (r *MemoryAuthRepository) UpdateAccessToken(_ context.Context, token identitymodels.AuthAccessToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token.Hash] = token
	return nil
}

func (r *MemoryAuthRepository) RevokeAccessToken(_ context.Context, tokenID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, token := range r.tokens {
		if token.ID == tokenID {
			now := time.Now().UTC()
			token.RevokedAt = &now
			token.UpdatedAt = now
			r.tokens[hash] = token
			return nil
		}
	}
	return ErrInvalidCredentials
}

type GormAuthRepository struct {
	db *gorm.DB
}

func NewGormAuthRepository(databaseURL string) (*GormAuthRepository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		return nil, err
	}
	return &GormAuthRepository{db: db}, nil
}

func (r *GormAuthRepository) CreateUser(ctx context.Context, user identitymodels.User) error {
	return r.db.WithContext(ctx).Create(&user).Error
}

func (r *GormAuthRepository) UpdateUser(ctx context.Context, user identitymodels.User) error {
	return r.db.WithContext(ctx).Save(&user).Error
}

func (r *GormAuthRepository) FindUserByEmail(ctx context.Context, email string) (*identitymodels.User, error) {
	var user identitymodels.User
	err := r.db.WithContext(ctx).Where("email = ?", normalizeEmail(email)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *GormAuthRepository) FindUserByID(ctx context.Context, id string) (*identitymodels.User, error) {
	var user identitymodels.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *GormAuthRepository) CreateIdentity(ctx context.Context, identity identitymodels.UserIdentity) error {
	return r.db.WithContext(ctx).Create(&identity).Error
}

func (r *GormAuthRepository) MarkIdentityEmailVerified(ctx context.Context, provider enum.AuthProvider, providerUserID string) error {
	return r.db.WithContext(ctx).
		Model(&identitymodels.UserIdentity{}).
		Where("provider = ? and provider_user_id = ?", provider, providerUserID).
		Updates(map[string]any{
			"email_verified": true,
			"updated_at":     time.Now().UTC(),
		}).Error
}

func (r *GormAuthRepository) CreateAccessToken(ctx context.Context, token identitymodels.AuthAccessToken) error {
	return r.db.WithContext(ctx).Create(&token).Error
}

func (r *GormAuthRepository) FindAccessTokenByHash(ctx context.Context, hash string) (*identitymodels.AuthAccessToken, error) {
	var token identitymodels.AuthAccessToken
	err := r.db.WithContext(ctx).First(&token, "hash = ?", hash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	return &token, err
}

func (r *GormAuthRepository) UpdateAccessToken(ctx context.Context, token identitymodels.AuthAccessToken) error {
	return r.db.WithContext(ctx).Save(&token).Error
}

func (r *GormAuthRepository) RevokeAccessToken(ctx context.Context, tokenID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&identitymodels.AuthAccessToken{}).Where("id = ?", tokenID).Updates(map[string]any{
		"revoked_at": now,
		"updated_at": now,
	}).Error
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomString(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func generateOTP() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	number := int(buf[0])<<24 | int(buf[1])<<16 | int(buf[2])<<8 | int(buf[3])
	if number < 0 {
		number = -number
	}
	return fmt.Sprintf("%06d", number%1000000)
}
