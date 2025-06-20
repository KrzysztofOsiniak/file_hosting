package user

import (
	db "backend/database"
	logdb "backend/logdatabase"
	"backend/util/logutil"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgerrcode"
)

type passwords struct {
	CurrentPassword string `json:"currentpassword"`
	NewPassword     string `json:"newpassword"`
}

func PatchPassword(w http.ResponseWriter, r *http.Request) {
	user := passwords{}
	// Limit reading the request body up to 1kB.
	// Input characters get automatically changed to � if they are invalid utf8:
	// Decode() acts the same as: https://pkg.go.dev/encoding/json#Unmarshal
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	// Rune count is used to count "characters" not bytes as would len() do.
	if utf8.RuneCountInString(user.CurrentPassword) > 25 || utf8.RuneCountInString(user.NewPassword) > 25 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(user.CurrentPassword) == 0 || utf8.RuneCountInString(user.NewPassword) > 25 {
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

	// Check if the current password matches.
	var hash string
	err = conn.QueryRow(ctx, "SELECT password_ FROM user_ WHERE id_ = $1", userID).Scan(&hash)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	match, err := argon2id.ComparePasswordAndHash(user.CurrentPassword, hash)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !match {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Hash is salted by default
	newHash, err := argon2id.CreateHash(user.NewPassword, argon2id.DefaultParams)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Change the password in the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "UPDATE user_ SET password_ = $1 WHERE id_ = $2", newHash, userID)
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

	if logdb.Pool == nil {
		return
	}
	logutil.Log(r.RemoteAddr, userID.(int), "", r.URL.Path, r.Method)
}
