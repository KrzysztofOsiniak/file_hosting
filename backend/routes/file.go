package routes

import (
	f "backend/controllers/file"
	m "backend/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Define routes with their middleware and controller.
func InitFile() *chi.Mux {
	fileRouter := chi.NewRouter()
	fileRouter.Handle("POST /upload-start", m.Auth(http.HandlerFunc(f.PostUploadStart)))
	fileRouter.Handle("POST /file-part", m.Auth(http.HandlerFunc(f.PostUploadPart)))
	fileRouter.Handle("POST /upload-complete", m.Auth(http.HandlerFunc(f.PostUploadComplete)))
	return fileRouter
}
