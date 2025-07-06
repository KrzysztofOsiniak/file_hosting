package user

import (
	db "backend/database"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

	// Get the sessions.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := conn.Query(ctx, "SELECT id_, expiry_date_, device_ FROM session_ WHERE user_id_ = $1", userID)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sessionArr)
}
