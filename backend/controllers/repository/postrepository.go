package repository

import (
	db "backend/database"
	"backend/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type repository struct {
	Visibility string
	Name       string
}

type repositoryResponse struct {
	ID int `json:"id"`
}

func PostRepository(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(types.ContextKey("id"))
	repo := repository{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&repo)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	if repo.Visibility != "public" && repo.Visibility != "private" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(repo.Name) > 35 || utf8.RuneCountInString(repo.Name) == 0 || utf8.RuneCountInString(repo.Visibility) == 0 {
		w.WriteHeader(http.StatusBadRequest)
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

	// Retry the transaction on serialization failure.
	res := repositoryResponse{}
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

		// Create the repository.
		err = tx.QueryRow(ctx, "CALL create_repository_(@userID, @name, @visibility, @repoID)",
			pgx.NamedArgs{"userID": userID, "name": repo.Name, "visibility": repo.Visibility, "repoID": nil}).Scan(&res.ID)
		var pgErr *pgconn.PgError
		ok := errors.As(err, &pgErr)
		if ok && pgErr.Code == pgerrcode.UniqueViolation {
			fmt.Println(err)
			w.WriteHeader(http.StatusConflict)
			return
		}
		if ok && pgErr.Code == pgerrcode.PrivilegeNotGranted {
			fmt.Println(err)
			w.WriteHeader(http.StatusForbidden)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
