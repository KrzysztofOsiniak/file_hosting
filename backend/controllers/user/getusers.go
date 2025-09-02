package user

import (
	db "backend/database"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type allUsersResponse struct {
	Users []getUser `json:"users"`
}

type getUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// Get an array of users that have the value of search in their username.
func GetUsers(w http.ResponseWriter, r *http.Request) {
	search := chi.URLParam(r, "username")

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

	// Get the users.
	rows, err := tx.Query(ctx, "SELECT id_, username_ FROM user_ WHERE LOWER(username_) LIKE '%' || LOWER($1) || '%' LIMIT 10", search)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Scan the rows into an array.
	userArr := allUsersResponse{}
	for rows.Next() {
		user := getUser{}
		err = rows.Scan(&user.ID, &user.Username)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		userArr.Users = append(userArr.Users, user)
	}
	if rows.Err() != nil {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userArr)
}
