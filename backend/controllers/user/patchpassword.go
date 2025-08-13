package user

import (
	db "backend/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type passwords struct {
	CurrentPassword string
	NewPassword     string
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
	if utf8.RuneCountInString(user.CurrentPassword) > 25 || utf8.RuneCountInString(user.NewPassword) > 25 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(user.CurrentPassword) == 0 || utf8.RuneCountInString(user.NewPassword) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the userID from the auth middleware.
	userID := r.Context().Value("id")

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

		// Check if the current password matches.
		var hash string
		err = tx.QueryRow(ctx, "SELECT password_ FROM user_ WHERE id_ = $1", userID).Scan(&hash)
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
		_, err = tx.Exec(ctx, "UPDATE user_ SET password_ = $1 WHERE id_ = $2", newHash, userID)
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

	w.WriteHeader(http.StatusOK)
}
