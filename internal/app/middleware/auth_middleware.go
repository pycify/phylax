package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	identitymodels "github.com/pycify/phylax/internal/app/models/identity"
	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
)

type contextKey string

const (
	userContextKey          contextKey = "phylax_user"
	tokenContextKey         contextKey = "phylax_access_token"
	projectAPIKeyContextKey contextKey = "phylax_project_api_key"
)

func Auth(service *identityservice.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			header := req.Header.Get("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if token == "" || token == header {
				writeAuthError(w, http.StatusUnauthorized, "Bearer token is required")
				return
			}
			user, accessToken, err := service.AuthenticateBearer(req.Context(), token)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}
			ctx := context.WithValue(req.Context(), userContextKey, user)
			ctx = context.WithValue(ctx, tokenContextKey, accessToken)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

func ProjectAPIKey(service *projectservice.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if service == nil {
				writeAuthError(w, http.StatusUnauthorized, "Project API key is required")
				return
			}
			header := req.Header.Get("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if token == "" || token == header {
				writeAuthError(w, http.StatusUnauthorized, "Project API key is required")
				return
			}
			apiKey, err := service.AuthenticateProjectAPIKey(req.Context(), token)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "Invalid project API key")
				return
			}
			ctx := context.WithValue(req.Context(), projectAPIKeyContextKey, apiKey)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

func CurrentUser(req *http.Request) (*identitymodels.User, bool) {
	user, ok := req.Context().Value(userContextKey).(*identitymodels.User)
	return user, ok
}

func CurrentAccessToken(req *http.Request) (*identitymodels.AuthAccessToken, bool) {
	token, ok := req.Context().Value(tokenContextKey).(*identitymodels.AuthAccessToken)
	return token, ok
}

func CurrentProjectAPIKey(req *http.Request) (*projectmodels.ProjectAPIKey, bool) {
	apiKey, ok := req.Context().Value(projectAPIKeyContextKey).(*projectmodels.ProjectAPIKey)
	return apiKey, ok
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   true,
		"message": message,
	})
}
