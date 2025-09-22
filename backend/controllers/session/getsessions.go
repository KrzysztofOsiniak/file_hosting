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

type allSessionsResponse struct {
	Sessions []session `json:"sessions"`
}

type session struct {
	ID         int       `json:"id"`
	ExpiryDate time.Time `json:"expiryDate"`
	Device     string    `json:"device"`
}

// Get all valid sessions for a user.
func GetSessions(w http.ResponseWriter, r *http.Request) {
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

	// Get the sessions.
	rows, err := tx.Query(ctx, "SELECT id_, expiry_date_, device_ FROM session_ WHERE user_id_ = $1", userID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Scan the rows into an array.
	sessionArr := allSessionsResponse{}
	for rows.Next() {
		session := session{}
		err = rows.Scan(&session.ID, &session.ExpiryDate, &session.Device)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		sessionArr.Sessions = append(sessionArr.Sessions, session)
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
	json.NewEncoder(w).Encode(sessionArr)
}
