package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/philiplambok/tudu/internal"
	v1 "github.com/philiplambok/tudu/pkg/openapi/v1"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	var body v1.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	req := CreateRequestDTO{Title: body.Title}
	if body.Description != nil {
		req.Description = *body.Description
	}
	if body.DueDate != nil {
		req.DueDate = body.DueDate
	}

	task, err := h.svc.Create(r.Context(), userID, req)
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to create task")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())
	status := r.URL.Query().Get("status")

	tasks, err := h.svc.List(r.Context(), userID, status)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	data := make([]v1.Task, len(tasks))
	for i := range tasks {
		data[i] = toV1Task(&tasks[i])
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskListResponse{Data: data})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to get task")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	var body v1.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	req := UpdateRequestDTO{
		Title:       body.Title,
		Description: body.Description,
		DueDate:     body.DueDate,
	}

	task, err := h.svc.Update(r.Context(), userID, id, req)
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to update task")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.svc.Complete(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to complete task")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskResponse{Data: toV1Task(task)})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListActivities(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	id, err := parseID(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid task id")
		return
	}

	activities, err := h.svc.ListActivities(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to list task activities")
		return
	}

	data := make([]v1.TaskActivity, len(activities))
	for i := range activities {
		data[i] = toV1TaskActivity(&activities[i])
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.TaskActivityListResponse{Data: data})
}

func toV1Task(t *TaskResponseDTO) v1.Task {
	task := v1.Task{
		Id:          t.ID,
		UserId:      t.UserID,
		Title:       t.Title,
		Status:      v1.TaskStatus(t.Status),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CompletedAt: t.CompletedAt,
		DueDate:     t.DueDate,
	}
	if t.Description != "" {
		task.Description = &t.Description
	}
	return task
}

func toV1TaskActivity(a *TaskActivityResponseDTO) v1.TaskActivity {
	return v1.TaskActivity{
		Id:        a.ID,
		TaskId:    a.TaskID,
		UserId:    a.UserID,
		Action:    v1.TaskActivityAction(a.Action),
		FieldName: a.FieldName,
		OldValue:  a.OldValue,
		NewValue:  a.NewValue,
		CreatedAt: a.CreatedAt,
	}
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	render.Status(r, status)
	render.JSON(w, r, v1.ErrorResponse{Error: msg})
}
