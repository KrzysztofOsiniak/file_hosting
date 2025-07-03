package test

import "net/http"

type integrationUser struct {
	Username string
	Password string
	Cookies  []*http.Cookie
}

type allUsers struct {
	Users []user
}

type user struct {
	ID       int
	Username string
	Role     string
}
