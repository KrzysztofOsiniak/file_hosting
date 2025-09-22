package middleware

import (
	db "backend/database"
	"backend/types"
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
		userID := r.Context().Value(types.ContextKey("id"))

		// Get a connection from the database and start a transaction.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		conn, err := db.GetConnection(ctx)
		defer conn.Release()
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly, DeferrableMode: pgx.Deferrable})
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// If commit is not run first this will rollback the transaction.
		defer tx.Rollback(ctx)

		// Check for admin role.
		var admin int
		err = tx.QueryRow(ctx, "SELECT 1 FROM user_ WHERE id_ = $1 AND role_ = 'admin'", userID).Scan(&admin)
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
		err = tx.Commit(ctx)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
