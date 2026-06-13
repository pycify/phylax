package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pycify/phylax/internal/app/controllers"
	attestationservice "github.com/pycify/phylax/internal/app/services/attestation"
	"github.com/pycify/phylax/internal/config"
	"github.com/pycify/phylax/internal/phylax"
	"github.com/pycify/phylax/internal/store"
	"github.com/stretchr/testify/require"
)

func TestRouterHealthAndOpenAPI(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t))
	defer server.Close()

	for _, path := range []string{"/v1/health", "/v1/openapi.json"} {
		response, err := http.Get(server.URL + path)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode)
		response.Body.Close()
	}
}

func TestRouterChallengeVerifyAndIntrospect(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t))
	defer server.Close()

	challenge := postJSONEnvelope[phylax.CreateChallengeResponse](t, server.URL+"/v1/apps/demo/challenges", map[string]any{
		"action":     "ai.ask",
		"install_id": "install_123",
	}, http.StatusCreated)

	verified := postJSONEnvelope[phylax.VerifyAttestationResponse](t, server.URL+"/v1/apps/demo/attestations/verify", map[string]any{
		"challenge_id": challenge.ChallengeID,
		"install_id":   "install_123",
		"token":        "mock",
	}, http.StatusOK)

	introspection := postJSONEnvelope[phylax.IntrospectResponse](t, server.URL+"/v1/sessions/introspect", map[string]any{
		"token": verified.Token,
	}, http.StatusOK)
	require.True(t, introspection.Active, introspection.Error)
}

func TestRouterValidatesChallengePayload(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t))
	defer server.Close()

	_ = postJSON[map[string]any](t, server.URL+"/v1/apps/demo/challenges", map[string]any{
		"action": "ai.ask",
	}, http.StatusUnprocessableEntity)
}

func TestRouterHandlesCORSPreflight(t *testing.T) {
	server := httptest.NewServer(newTestRouter(t))
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/v1/health", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, http.StatusNoContent, response.StatusCode)
	require.Equal(t, "*", response.Header.Get("Access-Control-Allow-Origin"))
	require.Contains(t, response.Header.Get("Access-Control-Allow-Methods"), http.MethodPost)
	require.Contains(t, response.Header.Get("Access-Control-Allow-Headers"), "Authorization")
}

func postJSON[T any](t *testing.T, url string, payload map[string]any, expectedStatus int) T {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	response, err := http.Post(url, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		responseBody, _ := io.ReadAll(response.Body)
		t.Fatalf("expected %d, got %d: %s", expectedStatus, response.StatusCode, responseBody)
	}
	var decoded T
	require.NoError(t, json.NewDecoder(response.Body).Decode(&decoded))
	return decoded
}

func postJSONEnvelope[T any](t *testing.T, url string, payload map[string]any, expectedStatus int) T {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	response, err := http.Post(url, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		responseBody, _ := io.ReadAll(response.Body)
		t.Fatalf("expected %d, got %d: %s", expectedStatus, response.StatusCode, responseBody)
	}
	var envelope struct {
		Error   bool            `json:"error"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&envelope))
	require.False(t, envelope.Error, envelope.Message)

	var decoded T
	require.NoError(t, json.Unmarshal(envelope.Data, &decoded))
	return decoded
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	service, err := phylax.NewService(config.Config{
		Environment: "development",
		Server:      config.ServerConfig{Addr: ":8080"},
		Storage:     config.StorageConfig{Driver: "memory"},
		Token: config.TokenConfig{
			Issuer:     "phylax-test",
			Secret:     "test-secret-with-more-than-32-bytes",
			TTLSeconds: 900,
		},
		Challenges: config.ChallengeConfig{TTLSeconds: 300},
	}, store.NewMemoryStore(), slog.New(slog.NewTextHandler(io.Discard, nil)), testAppResolver{apps: map[string]config.AppConfig{
		"demo": {
			ID:   "demo",
			Name: "Demo",
			Android: config.AndroidConfig{
				Enabled:               true,
				PackageName:           "com.example.demo",
				AllowMockVerification: true,
			},
		},
	}})
	require.NoError(t, err)
	_ = context.Background()
	attestationController := controllers.NewAttestationController(attestationservice.NewService(service))
	return NewRouter(attestationController, nil, nil, nil)
}

type testAppResolver struct {
	apps map[string]config.AppConfig
}

func (r testAppResolver) ResolveApp(_ context.Context, appID string) (config.AppConfig, error) {
	app, ok := r.apps[appID]
	if !ok {
		return config.AppConfig{}, phylax.ErrNotFound
	}
	return app, nil
}
