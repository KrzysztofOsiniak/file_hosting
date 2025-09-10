package user

import (
	db "backend/database"
	logdb "backend/logdatabase"
	m "backend/middleware"
	"backend/util/cookieutil"
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
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create the user and the user session in the database.
	var refreshToken string
	var userID int
	var userAgent string
	// Truncate the user agent to 255 characters.
	if length := utf8.RuneCountInString(r.UserAgent()); length > 254 {
		userAgent = r.UserAgent()[:254] + "..."
	} else {
		userAgent = r.UserAgent()[:length-1]
	}
	// Retry the transaction on serialization failure.
	var i int
	for i = 1; i <= 3; i++ {
		tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// If commit is not run first this will rollback the transaction.
		defer tx.Rollback(ctx)

		err = tx.QueryRow(ctx, "CALL create_user_and_session_($1, $2, $3, $4, $5)", user.Username, hash, userAgent, nil, nil).Scan(&refreshToken, &userID)
		var pgErr *pgconn.PgError
		ok := errors.As(err, &pgErr)
		if ok && pgErr.Code == pgerrcode.UniqueViolation {
			fmt.Println(err)
			w.WriteHeader(http.StatusConflict)
			return
		}
		if ok && pgErr.Code == pgerrcode.SerializationFailure {
			// End the transaction now to start another transaction.
			tx.Rollback(ctx)
			continue
		}
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = tx.Commit(ctx)
		ok = errors.As(err, &pgErr)
		if ok && pgErr.Code == pgerrcode.SerializationFailure {
			// End the transaction now to start another transaction.
			tx.Rollback(ctx)
			continue
		}
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		break
	}
	if i == 4 {
		fmt.Println("Failed serializing transaction after", i-1, "times")
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

	if logdb.Pool != nil {
		// Pass down user's username and id for deferred logging middleware.
		meta := r.Context().Value("meta").(*m.RequestMeta)
		meta.ID = userID
		meta.Username = user.Username
	}
}
