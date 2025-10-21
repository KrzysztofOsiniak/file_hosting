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
	repositoryRouter.Handle("GET /{id}", m.OptionalAuth(http.HandlerFunc(r.GetRepository)))
	repositoryRouter.Handle("GET /all-repositories", m.Auth(http.HandlerFunc(r.GetAllRepositories)))
	repositoryRouter.Handle("POST /", m.Auth(http.HandlerFunc(r.PostRepository)))
	repositoryRouter.Handle("DELETE /{id}", m.Auth(http.HandlerFunc(r.DeleteRepository)))
	repositoryRouter.Handle("PATCH /name", m.Auth(http.HandlerFunc(r.PatchName)))
	repositoryRouter.Handle("PATCH /visibility", m.Auth(http.HandlerFunc(r.PatchVisibility)))
	return repositoryRouter
}
