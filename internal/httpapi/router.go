package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pycify/phylax/internal/phylax"
)

type Router struct {
	service *phylax.Service
	mux     *http.ServeMux
}

func NewRouter(service *phylax.Service) http.Handler {
	router := &Router{
		service: service,
		mux:     http.NewServeMux(),
	}
	router.routes()
	return router.mux
}

func (r *Router) routes() {
	r.mux.HandleFunc("GET /v1/health", r.health)
	r.mux.HandleFunc("GET /v1/apps", r.apps)
	r.mux.HandleFunc("GET /v1/apps/", r.appProviders)
	r.mux.HandleFunc("POST /v1/challenges", r.createChallenge)
	r.mux.HandleFunc("POST /v1/attestations/verify", r.verifyAttestation)
	r.mux.HandleFunc("POST /v1/sessions/introspect", r.introspect)
}

func (r *Router) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "phylax",
	})
}

func (r *Router) apps(w http.ResponseWriter, _ *http.Request) {
	type appResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	apps := r.service.Apps()
	response := make([]appResponse, 0, len(apps))
	for _, app := range apps {
		response = append(response, appResponse{ID: app.ID, Name: app.Name})
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": response})
}

func (r *Router) appProviders(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/v1/apps/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "providers" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}

	statuses, err := r.service.ProviderStatuses(parts[0])
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": statuses})
}

func (r *Router) createChallenge(w http.ResponseWriter, req *http.Request) {
	var payload phylax.CreateChallengeRequest
	if err := readJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := r.service.CreateChallenge(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (r *Router) verifyAttestation(w http.ResponseWriter, req *http.Request) {
	var payload phylax.VerifyAttestationRequest
	if err := readJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := r.service.VerifyAttestation(req.Context(), payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (r *Router) introspect(w http.ResponseWriter, req *http.Request) {
	var payload phylax.IntrospectRequest
	if err := readJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, r.service.Introspect(req.Context(), payload.Token))
}

func readJSON(req *http.Request, out any) error {
	defer req.Body.Close()
	return json.NewDecoder(req.Body).Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":   true,
		"message": message,
	})
}
