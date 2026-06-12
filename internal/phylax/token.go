package phylax

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type tokenEnvelope struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type tokenClaims struct {
	Issuer    string    `json:"iss"`
	SessionID string    `json:"sid"`
	AppID     string    `json:"app_id"`
	Platform  Platform  `json:"platform"`
	Action    string    `json:"action"`
	InstallID string    `json:"install_id"`
	Subject   string    `json:"sub,omitempty"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

func signSessionToken(secret string, issuer string, session AttestationSession) (string, error) {
	headerBytes, err := json.Marshal(tokenEnvelope{Alg: "HS256", Typ: "PHYLAX"})
	if err != nil {
		return "", err
	}
	claimsBytes, err := json.Marshal(tokenClaims{
		Issuer:    issuer,
		SessionID: session.ID,
		AppID:     session.AppID,
		Platform:  session.Platform,
		Action:    session.Action,
		InstallID: session.InstallID,
		Subject:   session.Subject,
		IssuedAt:  session.IssuedAt,
		ExpiresAt: session.ExpiresAt,
	})
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := encodedHeader + "." + encodedClaims
	signature := sign(secret, signingInput)
	return signingInput + "." + signature, nil
}

func parseSessionToken(secret string, issuer string, token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	expectedSignature := sign(secret, parts[0]+"."+parts[1])
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return nil, errors.New("invalid token signature")
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims tokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, err
	}
	if claims.Issuer != issuer {
		return nil, errors.New("invalid token issuer")
	}
	if time.Now().UTC().After(claims.ExpiresAt) {
		return nil, errors.New("token expired")
	}
	return &claims, nil
}

func sign(secret string, input string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
