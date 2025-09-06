package file

import (
	db "backend/database"
	"backend/storage"
	"backend/types"
	t "backend/types"
	"backend/util/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type uploadFile struct {
	Key          string
	Size         int
	RepositoryID int
}

// Test uploading a ".." key.

func PostUploadStart(w http.ResponseWriter, r *http.Request) {
	f := uploadFile{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&f)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	if f.Size < config.MinFileSize {
		w.WriteHeader(http.StatusBadRequest)
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
	// If the file is being uploaded to the root of a repository, remove the returned "." character.
	if folderPath == "." {
		folderPath = ""
	}
	userID := r.Context().Value("id").(int)

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
	var data t.UploadStart
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

		// Check if user can upload the file.
		_, err = tx.Exec(ctx, "CALL prepare_file_(@repoID, @userID, @path, @folderPath, @size)",
			pgx.NamedArgs{"repoID": f.RepositoryID, "userID": userID, "path": f.Key, "folderPath": folderPath, "size": f.Size})
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

		data, err = storage.StartUpload(ctx, strconv.Itoa(userID)+"/"+strconv.Itoa(f.RepositoryID)+"/"+f.Key, f.Size)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Save the file to the db.
		err = tx.QueryRow(ctx, "INSERT INTO file_ VALUES (DEFAULT, @repoID, @userID, @path, @type, @size, @uploadID, NULL) RETURNING id_",
			pgx.NamedArgs{"repoID": f.RepositoryID, "userID": userID, "path": f.Key, "type": "file", "size": f.Size,
				"uploadID": data.UploadID}).Scan(&data.FileID)
		ok = errors.As(err, &pgErr)
		if ok && pgErr.Code == pgerrcode.UniqueViolation {
			fmt.Println(err)
			w.WriteHeader(http.StatusConflict)
			return
		}
		if ok && pgErr.Code == pgerrcode.SerializationFailure {
			fmt.Println("Retrying transaction...")
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
			fmt.Println("Retrying transaction...")
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
	res := types.UploadStartResponse{UploadParts: data.UploadParts, FileID: data.FileID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
