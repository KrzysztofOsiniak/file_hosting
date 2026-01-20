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
	memberRouter.Handle("DELETE /{id}", m.Auth(http.HandlerFunc(member.DeleteMember)))
	memberRouter.Handle("DELETE /leave/{id}", m.Auth(http.HandlerFunc(member.DeleteMemberLeave)))
	memberRouter.Handle("PATCH /permission", m.Auth(http.HandlerFunc(member.PatchPermission)))
	return memberRouter
}
