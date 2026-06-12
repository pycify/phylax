package phylax

import "time"

type Platform string

const (
	PlatformAndroid  Platform = "android"
	PlatformApple    Platform = "apple"
	PlatformFirebase Platform = "firebase"
)

type Challenge struct {
	ID         string    `json:"id"`
	AppID      string    `json:"app_id"`
	Platform   Platform  `json:"platform"`
	Action     string    `json:"action"`
	InstallID  string    `json:"install_id"`
	Nonce      string    `json:"nonce"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	ConsumedAt time.Time `json:"consumed_at,omitempty"`
}

type AttestationSession struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Platform  Platform  `json:"platform"`
	Action    string    `json:"action"`
	InstallID string    `json:"install_id"`
	Subject   string    `json:"subject,omitempty"`
	RiskLevel string    `json:"risk_level"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ProviderStatus struct {
	Platform string   `json:"platform"`
	Provider string   `json:"provider"`
	Enabled  bool     `json:"enabled"`
	Ready    bool     `json:"ready"`
	Issues   []string `json:"issues,omitempty"`
}

type Verdict struct {
	Platform       Platform       `json:"platform"`
	AppID          string         `json:"app_id"`
	InstallID      string         `json:"install_id"`
	Action         string         `json:"action"`
	RiskLevel      string         `json:"risk_level"`
	Provider       string         `json:"provider"`
	Claims         map[string]any `json:"claims"`
	AttestedAt     time.Time      `json:"attested_at"`
	ChallengeID    string         `json:"challenge_id"`
	ChallengeNonce string         `json:"-"`
}

type CreateChallengeRequest struct {
	AppID     string   `json:"app_id"`
	Platform  Platform `json:"platform"`
	Action    string   `json:"action"`
	InstallID string   `json:"install_id"`
	Subject   string   `json:"subject,omitempty"`
}

type CreateChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	AppID       string    `json:"app_id"`
	Platform    Platform  `json:"platform"`
	Action      string    `json:"action"`
	Nonce       string    `json:"nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type VerifyAttestationRequest struct {
	AppID       string         `json:"app_id"`
	Platform    Platform       `json:"platform"`
	ChallengeID string         `json:"challenge_id"`
	InstallID   string         `json:"install_id"`
	Subject     string         `json:"subject,omitempty"`
	Token       string         `json:"token,omitempty"`
	KeyID       string         `json:"key_id,omitempty"`
	Proof       map[string]any `json:"proof,omitempty"`
}

type VerifyAttestationResponse struct {
	Token     string             `json:"token"`
	Session   AttestationSession `json:"session"`
	Verdict   Verdict            `json:"verdict"`
	ExpiresAt time.Time          `json:"expires_at"`
}

type IntrospectRequest struct {
	Token string `json:"token"`
}

type IntrospectResponse struct {
	Active  bool                `json:"active"`
	Session *AttestationSession `json:"session,omitempty"`
	Error   string              `json:"error,omitempty"`
}
