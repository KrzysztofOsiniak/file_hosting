package user

import (
	db "backend/database"
	"context"
	"fmt"
	"net/http"
	"time"
)

func PostLogout(w http.ResponseWriter, r *http.Request) {
	// Get the userID and refresh token from the auth middleware.
	userID := r.Context().Value("id")
	refreshToken := r.Context().Value("session")

	// Get a connection from the database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Delete the session from the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// No need to check for returned rows, therefore Exec() is used.
	_, err = conn.Exec(ctx, "DELETE FROM session_ WHERE user_id_ = $1 AND token_ = $2", userID, refreshToken)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Create an empty cookie to unset the current one.
	cookie := http.Cookie{
		Name:     "file_hosting",
		Value:    "",
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
}
