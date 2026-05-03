package user

import (
	"github.com/go-chi/chi/v5"
	"github.com/philiplambok/tudu/pkg/avatar"
	"gorm.io/gorm"
)

type Endpoint struct {
	handler *Handler
}

func NewEndpoint(db *gorm.DB, avatarProvider avatar.Provider, jwtSecret string) *Endpoint {
	repo := NewRepository(db)
	svc := NewService(repo, avatarProvider, jwtSecret)
	return &Endpoint{handler: NewHandler(svc)}
}

// AuthRoutes returns public auth routes (no JWT required).
func (e *Endpoint) AuthRoutes() *chi.Mux {
	r := chi.NewMux()
	r.Post("/register", e.handler.Register)
	r.Post("/login", e.handler.Login)
	return r
}

// MeRoutes returns the protected /me route (JWT required).
func (e *Endpoint) MeRoutes() *chi.Mux {
	r := chi.NewMux()
	r.Get("/me", e.handler.Me)
	return r
}
