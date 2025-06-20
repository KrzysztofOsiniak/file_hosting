package routes

import (
	u "backend/controllers/user"
	m "backend/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Define routes with their middleware and controller.
func InitUser() *chi.Mux {
	userRouter := chi.NewRouter()
	userRouter.Handle("POST /", http.HandlerFunc(u.PostUser))
	userRouter.Handle("POST /login", http.HandlerFunc(u.PostLogin))
	userRouter.Handle("POST /logout", m.Auth(http.HandlerFunc(u.PostLogout)))
	userRouter.Handle("DELETE /", m.Auth(http.HandlerFunc(u.DeleteUser)))
	userRouter.Handle("PATCH /username", m.Auth(http.HandlerFunc(u.PatchUsername)))
	userRouter.Handle("PATCH /password", m.Auth(http.HandlerFunc(u.PatchPassword)))
	return userRouter
}
