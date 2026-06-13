package helper

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/pycify/phylax/internal/app/middleware"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
	"github.com/pycify/phylax/internal/app/types/response"
)

func RequireUserID(w http.ResponseWriter, req *http.Request) (string, bool) {
	user, ok := middleware.CurrentUser(req)
	if !ok || user == nil {
		WriteError(w, http.StatusUnauthorized, "Authenticated user missing from context")
		return "", false
	}
	return user.ID, true
}

func DecodeAndValidate(w http.ResponseWriter, req *http.Request, validate *validator.Validate, out any) bool {
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(out); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return false
	}
	if err := validate.Struct(out); err != nil {
		WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return false
	}
	return true
}

func WriteServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, projectservice.ErrProjectNotFound) || errors.Is(err, projectservice.ErrAppNotFound) {
		WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	WriteError(w, http.StatusUnprocessableEntity, err.Error())
}

func WriteJSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response.APIResponse[T]{
		Error:   false,
		Message: "success",
		Data:    &data,
	})
}

func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response.ErrorResponse{
		Error:   true,
		Message: message,
	})
}
