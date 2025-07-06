package user

import (
	db "backend/database"
	logdb "backend/logdatabase"
	m "backend/middleware"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgerrcode"
)

func PatchUsername(w http.ResponseWriter, r *http.Request) {
	user := user{}
	// Limit reading the request body up to 1kB.
	// Input characters get automatically changed to � if they are invalid utf8:
	// Decode() acts the same as: https://pkg.go.dev/encoding/json#Unmarshal
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	user.Username = strings.TrimSpace(user.Username)
	// Rune count is used to count "characters" not bytes as would len() do.
	if utf8.RuneCountInString(user.Username) > 25 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(user.Username) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

	// Change the username in the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "UPDATE user_ SET username_ = $1 WHERE id_ = $2", user.Username, userID)
	if err != nil && strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
		fmt.Println(err)
		w.WriteHeader(http.StatusConflict)
		return
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	if logdb.Pool != nil {
		// Pass down user's username for deferred logging middleware.
		var meta *m.RequestMeta
		meta = r.Context().Value("meta").(*m.RequestMeta)
		meta.Username = user.Username
	}
}
