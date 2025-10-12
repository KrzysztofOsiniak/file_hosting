package user

import (
	db "backend/database"
	"backend/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type userAccount struct {
	Username   string `json:"username"`
	Role       string `json:"role"`
	SpaceTaken int    `json:"spaceTaken"`
	Space      int    `json:"space"`
}

// Get the user's account details like:
// username, role, space taken, all space.
func GetAccount(w http.ResponseWriter, r *http.Request) {
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

	// Get the user's details.
	var user userAccount
	err = tx.QueryRow(ctx, `SELECT username_, role_, COALESCE((SELECT SUM(size_) FROM file_ WHERE user_id_=user_.id_), 0), space_ FROM 
		user_ WHERE id_ = $1`, userID).Scan(&user.Username, &user.Role, &user.SpaceTaken, &user.Space)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
