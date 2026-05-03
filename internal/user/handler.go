package user

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body v1.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Register(r.Context(), RegisterRequestDTO{
		Email:    string(body.Email),
		Password: body.Password,
	})
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		if errors.Is(err, ErrEmailConflict) {
			writeError(w, r, http.StatusConflict, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to register")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, v1.AuthResponse{
		Token: resp.Token,
		Data:  v1.UserData{User: toV1User(&resp.User)},
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body v1.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Login(r.Context(), LoginRequestDTO{
		Email:    string(body.Email),
		Password: body.Password,
	})
	if err != nil {
		var ve *ValidationError
		if errors.As(err, &ve) {
			writeError(w, r, http.StatusUnprocessableEntity, ve.Error())
			return
		}
		if errors.Is(err, ErrInvalidCreds) {
			writeError(w, r, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to login")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.AuthResponse{
		Token: resp.Token,
		Data:  v1.UserData{User: toV1User(&resp.User)},
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := internal.UserIDFromContext(r.Context())

	u, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "failed to get user")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, v1.UserResponse{Data: v1.UserData{User: toV1User(u)}})
}

func toV1User(u *UserResponseDTO) v1.User {
	return v1.User{
		Id:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	render.Status(r, status)
	render.JSON(w, r, v1.ErrorResponse{Error: msg})
}
