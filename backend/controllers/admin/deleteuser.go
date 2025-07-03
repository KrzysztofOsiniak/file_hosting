package admin

import (
	db "backend/database"
	logdb "backend/logdatabase"
	"backend/util/logutil"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Delete a user as an admin.
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get the userID from the auth middleware.
	userID := r.Context().Value("id")
	deleteID := chi.URLParam(r, "id")

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

	// Delete the user from the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "DELETE from user_ WHERE id_ = $1", deleteID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	if logdb.Pool == nil {
		return
	}
	logutil.Log(r.RemoteAddr, userID.(int), "", r.URL.Path, r.Method)
}
