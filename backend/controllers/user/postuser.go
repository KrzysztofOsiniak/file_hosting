package user

import (
	db "backend/database"
	logdb "backend/logdatabase"
	"backend/util/cookieutil"
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

// Called when creating an account.
// Create a new user and the user session in the database and set a new JWT as a cookie.
func PostUser(w http.ResponseWriter, r *http.Request) {
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
	// Hash is salted by default
	hash, err := argon2id.CreateHash(user.Password, argon2id.DefaultParams)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
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

	// Create the user and the user session in the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var refreshToken string
	var userID int
	var userAgent string
	// Truncate the user agent to 255 characters.
	if length := utf8.RuneCountInString(r.UserAgent()); length > 254 {
		userAgent = r.UserAgent()[:254] + "..."
	} else {
		userAgent = r.UserAgent()[:length-1]
	}
	// The two last arguments have to be specified (can be empty), to be passed into the output parameters:
	// create_user_and_session_(username TEXT, password TEXT, device TEXT, OUT token UUID, OUT user_id INT)
	err = conn.QueryRow(ctx, "CALL create_user_and_session_($1, $2, $3, $4, $5)", user.Username, hash, userAgent, nil, nil).Scan(&refreshToken, &userID)
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
