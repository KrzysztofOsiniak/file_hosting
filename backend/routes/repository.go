package routes

import (
	r "backend/controllers/repository"
	m "backend/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Define routes with their middleware and controller.
func InitRepository() *chi.Mux {
	repositoryRouter := chi.NewRouter()
	repositoryRouter.Handle("POST /", m.Auth(http.HandlerFunc(r.PostRepository)))
	return repositoryRouter
}
