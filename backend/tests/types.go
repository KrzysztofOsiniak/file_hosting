package test

import (
	"backend/types"
	"net/http"
)

type integrationUser struct {
	Username     string
	Password     string
	Cookies      []*http.Cookie
	RepositoryID int    // Used to delete/modify a repository.
	FolderPath   string // Used for uploading files/folders and is added to the start of their key/path.
	FolderID     int    // Used to delete/modify a folder.
	FileID       int    // Used to delete/modify a file.
	MemberID     int    // Used to delete/modify a member, this is set for secondTestUser after subtest_postmember.
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

type uploadFile struct {
	Key          string
	Size         int
	RepositoryID int
}

type uploadCompleteRequest struct {
	ID int
}

type filePartRequest struct {
	types.CompletePart
	FileID int
}
