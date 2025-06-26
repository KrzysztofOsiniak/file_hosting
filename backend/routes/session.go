package routes

import (
	s "backend/controllers/session"
	m "backend/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Define routes with their middleware and controller.
func InitSession() *chi.Mux {
	sessionRouter := chi.NewRouter()
	sessionRouter.Handle("GET /all", m.Auth(http.HandlerFunc(s.GetSessions)))
	sessionRouter.Handle("DELETE /all", m.Auth(http.HandlerFunc(s.DeleteSessions)))
	sessionRouter.Handle("DELETE /{id}", m.Auth(http.HandlerFunc(s.DeleteSession)))
	return sessionRouter
}
