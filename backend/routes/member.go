package routes

import (
	member "backend/controllers/member"
	m "backend/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Define routes with their middleware and controller.
func InitMember() *chi.Mux {
	memberRouter := chi.NewRouter()
	memberRouter.Handle("POST /", m.Auth(http.HandlerFunc(member.PostMember)))
	return memberRouter
}
