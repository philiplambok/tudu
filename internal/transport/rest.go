package transport

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/philiplambok/tudu/internal"
	"github.com/philiplambok/tudu/internal/swagger"
	"github.com/philiplambok/tudu/internal/task"
	"github.com/philiplambok/tudu/internal/user"
	"gorm.io/gorm"
)

type Server struct {
	mux *chi.Mux
	cfg internal.Config
}

func NewServer(cfg internal.Config, db *gorm.DB) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	swagger.Register(r)

	userEndpoint := user.NewEndpoint(db, cfg.JWT.Secret)
	r.Mount("/v1/auth", userEndpoint.AuthRoutes())

	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware(cfg.JWT.Secret))
		r.Mount("/v1/users", userEndpoint.MeRoutes())
		r.Mount("/v1/tasks", task.NewEndpoint(db).Routes())
	})

	return &Server{mux: r, cfg: cfg}
}

func (s *Server) Start() error {
	addr := ":" + s.cfg.HTTPServer.Port
	slog.Info("tudu listening", "addr", addr)
	return http.ListenAndServe(addr, s.mux)
}

func jwtMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			sub, err := claims.GetSubject()
			if err != nil {
				http.Error(w, `{"error":"invalid token subject"}`, http.StatusUnauthorized)
				return
			}

			var userID int64
			if _, err := fmt.Sscanf(sub, "%d", &userID); err != nil {
				http.Error(w, `{"error":"invalid token subject"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(internal.WithUserID(r.Context(), userID)))
		})
	}
}
