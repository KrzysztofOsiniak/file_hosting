package user

import (
	db "backend/database"
	"context"
	"fmt"
	"net/http"
	"time"
)

// Delete all user's sessions.
func DeleteSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("id")

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

	// Delete all user's sessions from the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "DELETE from session_ WHERE user_id_ = $1", userID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
