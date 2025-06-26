package user

import (
	db "backend/database"
	logdb "backend/logdatabase"
	"backend/util/cookieutil"
	"backend/util/logutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
)

func PostLogin(w http.ResponseWriter, r *http.Request) {
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
	if utf8.RuneCountInString(user.Username) > 25 || utf8.RuneCountInString(user.Password) > 60 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(user.Username) == 0 || utf8.RuneCountInString(user.Password) == 0 {
		w.WriteHeader(http.StatusBadRequest)
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

	// Check if the user exists.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var userID int
	var hash string
	// Get the user's id in case the credentials match.
	err = conn.QueryRow(ctx, "SELECT id_, password_ FROM user_ WHERE username_ = $1", user.Username).Scan(&userID, &hash)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		fmt.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check if the credentials match.
	match, err := argon2id.ComparePasswordAndHash(user.Password, hash)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !match {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create a new session.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var refreshToken string
	var userAgent string
	// Truncate the user agent to 255 characters.
	if length := utf8.RuneCountInString(r.UserAgent()); length > 254 {
		userAgent = r.UserAgent()[:254] + "..."
	} else {
		userAgent = r.UserAgent()[:length-1]
	}
	err = conn.QueryRow(ctx, "INSERT INTO session_ VALUES (DEFAULT, $1, GEN_RANDOM_UUID(), CURRENT_TIMESTAMP(0) + INTERVAL '14 day', $2) RETURNING token_", userID, userAgent).Scan(&refreshToken)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create a cookie to be sent.
	newCookie, err := cookieutil.CreateJWTCookie(userID, refreshToken)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, newCookie)
	w.WriteHeader(http.StatusOK)

	if logdb.Pool == nil {
		return
	}
	logutil.Log(r.RemoteAddr, userID, user.Username, r.URL.Path, r.Method)
}
