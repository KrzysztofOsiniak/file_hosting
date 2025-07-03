package routes

import (
	a "backend/controllers/admin"
	m "backend/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Define routes with their middleware and controller.
func InitAdmin() *chi.Mux {
	adminRouter := chi.NewRouter()
	adminRouter.Handle("GET /users/{username}", m.Auth(m.Admin(http.HandlerFunc(a.GetUsers))))
	adminRouter.Handle("DELETE /user/{id}", m.Auth(m.Admin(http.HandlerFunc(a.DeleteUser))))
	adminRouter.Handle("PATCH /user/role/{id}", m.Auth(m.Admin(http.HandlerFunc(a.PatchUserRole))))
	return adminRouter
}
