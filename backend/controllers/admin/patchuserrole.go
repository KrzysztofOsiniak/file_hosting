package admin

import (
	db "backend/database"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type userRole struct {
	Role string
}

// Patch a user's role to one out of the three possible.
func PatchUserRole(w http.ResponseWriter, r *http.Request) {
	patchID := chi.URLParam(r, "id")
	user := userRole{}
	// Limit reading the request body up to 1kB.
	// Input characters get automatically changed to � if they are invalid utf8:
	// Decode() acts the same as: https://pkg.go.dev/encoding/json#Unmarshal
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

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

	// Change user's role in the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "UPDATE user_ SET role_ = $1 WHERE id_ = $2", user.Role, patchID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
