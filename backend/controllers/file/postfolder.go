package file

import (
	db "backend/database"
	"backend/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type folder struct {
	Key          string
	RepositoryID int
}

type folderResponse struct {
	ID int `json:"id"`
}

func PostFolder(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(types.ContextKey("id")).(int)
	f := folder{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&f)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	f.Key = path.Clean(f.Key)
	if f.Key == "." {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Check if the cleaned key (path) is valid.
	runes := []rune(f.Key)
	if string(runes[0]) == "/" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	folderPath := path.Dir(f.Key)
	// If the folder is being uploaded to the root of a repository, remove the returned "." character.
	if folderPath == "." {
		folderPath = ""
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
	var res folderResponse
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

		// Check if the user can add this folder.
		_, err = tx.Exec(ctx, "CALL prepare_folder_(@repoID, @userID, @path, @folderPath)",
			pgx.NamedArgs{"repoID": f.RepositoryID, "userID": userID, "path": f.Key, "folderPath": folderPath})
		var pgErr *pgconn.PgError
		ok := errors.As(err, &pgErr)
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

		// Save the folder to the db.
		err = tx.QueryRow(ctx, "INSERT INTO file_ VALUES (DEFAULT, @repoID, @userID, @path, @type, 0, NULL, CURRENT_TIMESTAMP(0)) RETURNING id_",
			pgx.NamedArgs{"repoID": f.RepositoryID, "userID": userID, "path": f.Key, "type": "folder"}).Scan(&res.ID)
		ok = errors.As(err, &pgErr)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
