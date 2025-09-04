package test

import (
	"backend/types"
	"net/http"
)

type integrationUser struct {
	Username     string
	Password     string
	Cookies      []*http.Cookie
	RepositoryID int
	FolderPath   string
}

type allUsers struct {
	Users []user
}

type user struct {
	ID       int
	Username string
	Role     string
}

type member struct {
	UserID       int
	Permission   string
	RepositoryID int
}

type uploadPart struct {
	URL  string
	Part int
}

type uploadFile struct {
	Key          string
	Size         int
	RepositoryID int
}

type uploadCompleteRequest struct {
	FileID int
}

type filePartRequest struct {
	types.CompletePart
	FileID int
}
