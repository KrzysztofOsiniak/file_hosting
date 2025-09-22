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
	fileRouter.Handle("GET /{fileID}", m.OptionalAuth(http.HandlerFunc(f.GetDownload)))
	fileRouter.Handle("POST /folder", m.Auth(http.HandlerFunc(f.PostFolder)))
	fileRouter.Handle("POST /upload-start", m.Auth(http.HandlerFunc(f.PostUploadStart)))
	fileRouter.Handle("POST /file-part", m.Auth(http.HandlerFunc(f.PostUploadPart)))
	fileRouter.Handle("POST /upload-complete", m.Auth(http.HandlerFunc(f.PostUploadComplete)))
	fileRouter.Handle("POST /upload-resume", m.Auth(http.HandlerFunc(f.PostResumeUpload)))
	fileRouter.Handle("DELETE /folder/{id}", m.Auth(http.HandlerFunc(f.DeleteFolder)))
	fileRouter.Handle("DELETE /{id}", m.Auth(http.HandlerFunc(f.DeleteFile)))
	fileRouter.Handle("DELETE /in-progress/{id}", m.Auth(http.HandlerFunc(f.DeleteInProgress)))
	fileRouter.Handle("PATCH /name", m.Auth(http.HandlerFunc(f.PatchFileName)))
	fileRouter.Handle("PATCH /folder/name", m.Auth(http.HandlerFunc(f.PatchFolderName)))
	return fileRouter
}
