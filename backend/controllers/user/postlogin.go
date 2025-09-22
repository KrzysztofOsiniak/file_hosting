package user

import (
	db "backend/database"
	logdb "backend/logdatabase"
	m "backend/middleware"
	"backend/types"
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
	if utf8.RuneCountInString(user.Username) > 25 || utf8.RuneCountInString(user.Password) > 60 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(user.Username) == 0 || utf8.RuneCountInString(user.Password) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

	// Check if the user exists.
	var userID int
	var hash string
	// Get the user's id in case the credentials match.
	err = tx.QueryRow(ctx, "SELECT id_, password_ FROM user_ WHERE LOWER(username_) = LOWER($1)", user.Username).Scan(&userID, &hash)
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
	err = tx.Commit(ctx)
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
	var refreshToken string
	var userAgent string
	// Truncate the user agent to 255 unicode points.
	if length := utf8.RuneCountInString(r.UserAgent()); length > 255 {
		userAgent = string([]rune(r.UserAgent())[:254])
	} else {
		userAgent = r.UserAgent()
	}
	var i int
	// Retry the transaction on serialization failure.
	for i = 1; i <= 3; i++ {
		tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// If commit is not run first this will rollback the transaction.
		defer tx.Rollback(ctx)

		err = tx.QueryRow(ctx, "INSERT INTO session_ VALUES (DEFAULT, $1, GEN_RANDOM_UUID(), CURRENT_TIMESTAMP(0) + INTERVAL '14 day', $2) RETURNING token_", userID, userAgent).Scan(&refreshToken)
		var pgErr *pgconn.PgError
		ok := errors.As(err, &pgErr)
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
		meta := r.Context().Value(types.ContextKey("meta")).(*m.RequestMeta)
		meta.ID = userID
		meta.Username = user.Username
	}
}
