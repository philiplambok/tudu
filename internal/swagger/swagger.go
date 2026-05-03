package swagger

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	v1 "github.com/philiplambok/tudu/pkg/openapi/v1"
	httpSwagger "github.com/swaggo/http-swagger"
)

func Register(r chi.Router) {
	r.Get("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		s, err := v1.GetSpec()
		if err != nil {
			http.Error(w, "failed to load spec", http.StatusInternalServerError)
			return
		}
		b, err := s.MarshalJSON()
		if err != nil {
			http.Error(w, "failed to marshal spec", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})

	r.Handle("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
	))
}
