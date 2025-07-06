package admin

import (
	db "backend/database"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type allUsersResponse struct {
	Users []user `json:"users"`
}

type user struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Get an array of users that have the value of search in their username.
func GetUsers(w http.ResponseWriter, r *http.Request) {
	search := chi.URLParam(r, "username")

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

	// Get the users.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// Use || for string concatenation.
	rows, err := conn.Query(ctx, "SELECT id_, username_, role_ FROM user_ WHERE LOWER(username_) LIKE '%' || LOWER($1) || '%' LIMIT 10", search)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Scan the rows into an array.
	userArr := allUsersResponse{}
	for rows.Next() {
		user := user{}
		err = rows.Scan(&user.ID, &user.Username, &user.Role)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userArr)
}
