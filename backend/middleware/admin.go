package middleware

import (
	db "backend/database"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// Check if the user is an admin.
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the userID from the auth middleware.
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

		// Check for admin role.
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		var admin int
		err = conn.QueryRow(ctx, "SELECT 1 FROM user_ WHERE id_ = $1 AND role_ = 'admin'", userID).Scan(&admin)
		if errors.Is(err, pgx.ErrNoRows) {
			fmt.Println(err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
