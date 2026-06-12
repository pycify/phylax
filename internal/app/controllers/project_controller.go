package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/pycify/phylax/internal/app/middleware"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
	projectdto "github.com/pycify/phylax/internal/app/types/dto/projects"
	"github.com/pycify/phylax/internal/app/types/response"
	projectvalidators "github.com/pycify/phylax/internal/app/validators/projects"
)

type ProjectController struct {
	service  *projectservice.Service
	validate *validator.Validate
}

func NewProjectController(service *projectservice.Service) *ProjectController {
	return &ProjectController{
		service:  service,
		validate: validator.New(),
	}
}

// CreateProject godoc
//
//	@Summary	Create project
//	@Tags		Projects
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		payload				body		projects.CreateProjectRequest		true	"Project payload"
//	@Success	201					{object}	response.ProjectResponse
//	@Failure	400					{object}	response.ErrorResponse
//	@Failure	422					{object}	response.ErrorResponse
//	@Router		/v1/projects [post]
func (c *ProjectController) CreateProject(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	var payload projectvalidators.CreateProjectRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	project, err := c.service.CreateProject(req.Context(), userID, payload)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, projectdto.ProjectResponse{Project: projectdto.NewProjectDTO(project)})
}

// ListProjects godoc
//
//	@Summary	List customer projects
//	@Tags		Projects
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200					{object}	response.ProjectsResponse
//	@Failure	400					{object}	response.ErrorResponse
//	@Router		/v1/projects [get]
func (c *ProjectController) ListProjects(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	projects, err := c.service.ListProjects(req.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response := make([]projectdto.ProjectDTO, 0, len(projects))
	for _, project := range projects {
		response = append(response, projectdto.NewProjectDTO(project))
	}
	writeJSON(w, http.StatusOK, projectdto.ProjectsResponse{Projects: response})
}

// GetProject godoc
//
//	@Summary	Get project
//	@Tags		Projects
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string	true	"Project id"
//	@Success	200					{object}	response.ProjectResponse
//	@Failure	404					{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id} [get]
func (c *ProjectController) GetProject(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	project, err := c.service.GetProject(req.Context(), userID, chi.URLParam(req, "project_id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectdto.ProjectResponse{Project: projectdto.NewProjectDTO(*project)})
}

// UpdateProject godoc
//
//	@Summary	Update project
//	@Tags		Projects
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string								true	"Project id"
//	@Param		payload				body		projects.UpdateProjectRequest		true	"Project payload"
//	@Success	200					{object}	response.ProjectResponse
//	@Failure	404					{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id} [patch]
func (c *ProjectController) UpdateProject(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	var payload projectvalidators.UpdateProjectRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	project, err := c.service.UpdateProject(req.Context(), userID, chi.URLParam(req, "project_id"), payload)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectdto.ProjectResponse{Project: projectdto.NewProjectDTO(*project)})
}

// DeleteProject godoc
//
//	@Summary	Delete project
//	@Tags		Projects
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path	string	true	"Project id"
//	@Success	204
//	@Failure	404	{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id} [delete]
func (c *ProjectController) DeleteProject(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	if err := c.service.DeleteProject(req.Context(), userID, chi.URLParam(req, "project_id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateProjectApp godoc
//
//	@Summary	Create project app
//	@Tags		Project Apps
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string									true	"Project id"
//	@Param		payload				body		projects.CreateProjectAppRequest		true	"App payload"
//	@Success	201					{object}	response.ProjectAppResponse
//	@Failure	422					{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id}/apps [post]
func (c *ProjectController) CreateProjectApp(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	var payload projectvalidators.CreateProjectAppRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	app, err := c.service.CreateApp(req.Context(), userID, chi.URLParam(req, "project_id"), payload)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, projectdto.ProjectAppResponse{App: projectdto.NewProjectAppDTO(app)})
}

// ListProjectApps godoc
//
//	@Summary	List project apps
//	@Tags		Project Apps
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string	true	"Project id"
//	@Success	200					{object}	response.ProjectAppsResponse
//	@Router		/v1/projects/{project_id}/apps [get]
func (c *ProjectController) ListProjectApps(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	apps, err := c.service.ListApps(req.Context(), userID, chi.URLParam(req, "project_id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response := make([]projectdto.ProjectAppDTO, 0, len(apps))
	for _, app := range apps {
		response = append(response, projectdto.NewProjectAppDTO(app))
	}
	writeJSON(w, http.StatusOK, projectdto.ProjectAppsResponse{Apps: response})
}

// GetProjectApp godoc
//
//	@Summary	Get project app
//	@Tags		Project Apps
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string	true	"Project id"
//	@Param		app_id				path		string	true	"App id"
//	@Success	200					{object}	response.ProjectAppResponse
//	@Failure	404					{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id}/apps/{app_id} [get]
func (c *ProjectController) GetProjectApp(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	app, err := c.service.GetApp(req.Context(), userID, chi.URLParam(req, "project_id"), chi.URLParam(req, "app_id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectdto.ProjectAppResponse{App: projectdto.NewProjectAppDTO(*app)})
}

// UpdateProjectApp godoc
//
//	@Summary	Update project app
//	@Tags		Project Apps
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string									true	"Project id"
//	@Param		app_id				path		string									true	"App id"
//	@Param		payload				body		projects.UpdateProjectAppRequest		true	"App payload"
//	@Success	200					{object}	response.ProjectAppResponse
//	@Failure	404					{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id}/apps/{app_id} [patch]
func (c *ProjectController) UpdateProjectApp(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	var payload projectvalidators.UpdateProjectAppRequest
	if !decodeAndValidate(w, req, c.validate, &payload) {
		return
	}
	app, err := c.service.UpdateApp(req.Context(), userID, chi.URLParam(req, "project_id"), chi.URLParam(req, "app_id"), payload)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectdto.ProjectAppResponse{App: projectdto.NewProjectAppDTO(*app)})
}

// DeleteProjectApp godoc
//
//	@Summary	Soft delete project app
//	@Tags		Project Apps
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path	string	true	"Project id"
//	@Param		app_id				path	string	true	"App id"
//	@Success	204
//	@Failure	404	{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id}/apps/{app_id} [delete]
func (c *ProjectController) DeleteProjectApp(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	if err := c.service.DeleteApp(req.Context(), userID, chi.URLParam(req, "project_id"), chi.URLParam(req, "app_id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetProjectAppAttestationStatus godoc
//
//	@Summary	Get project app attestation status
//	@Tags		Project Apps
//	@Produce	json
//	@Security	BearerAuth
//	@Param		project_id			path		string	true	"Project id"
//	@Param		app_id				path		string	true	"App id"
//	@Success	200					{object}	response.AppAttestationStatusResponse
//	@Failure	404					{object}	response.ErrorResponse
//	@Router		/v1/projects/{project_id}/apps/{app_id}/attestation-status [get]
func (c *ProjectController) GetProjectAppAttestationStatus(w http.ResponseWriter, req *http.Request) {
	userID, ok := requireUserID(w, req)
	if !ok {
		return
	}
	status, err := c.service.GetAppAttestationStatus(req.Context(), userID, chi.URLParam(req, "project_id"), chi.URLParam(req, "app_id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectdto.AppAttestationStatusResponse{
		Status: projectdto.NewAppAttestationStatusDTO(*status),
	})
}

func requireUserID(w http.ResponseWriter, req *http.Request) (string, bool) {
	user, ok := middleware.CurrentUser(req)
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "Authenticated user missing from context")
		return "", false
	}
	return user.ID, true
}

func decodeAndValidate(w http.ResponseWriter, req *http.Request, validate *validator.Validate, out any) bool {
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}
	if err := validate.Struct(out); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, projectservice.ErrProjectNotFound) || errors.Is(err, projectservice.ErrAppNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusUnprocessableEntity, err.Error())
}

func writeJSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response.APIResponse[T]{
		Error:   false,
		Message: "success",
		Data:    &data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response.ErrorResponse{
		Error:   true,
		Message: message,
	})
}
