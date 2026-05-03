package task

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Endpoint struct {
	handler *Handler
}

func NewEndpoint(db *gorm.DB) *Endpoint {
	repo := NewRepository(db)
	svc := NewService(repo)
	return &Endpoint{handler: NewHandler(svc)}
}

func (e *Endpoint) Routes() *chi.Mux {
	r := chi.NewMux()
	r.Post("/", e.handler.Create)
	r.Get("/", e.handler.List)
	r.Get("/{id}", e.handler.Get)
	r.Patch("/{id}", e.handler.Update)
	r.Post("/{id}/complete", e.handler.Complete)
	r.Delete("/{id}", e.handler.Delete)
	return r
}
