package file

import (
	db "backend/database"
	"backend/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

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

	// Get a connection from the database and start a transaction.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// If commit is not run first this will rollback the transaction.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		tx.Rollback(ctx)
	}()

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	userID := r.Context().Value("id")
	_, err = tx.Exec(ctx, "CALL prepare_file_(@repoID, @userID, @path, @folderPath, @size)",
		pgx.NamedArgs{"repoID": f.RepositoryID, "userID": userID, "path": f.Key, "folderPath": folderPath, "size": f.Size})
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok && pgErr.Code == "01007" {
		fmt.Println(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := storage.StartUpload(strconv.Itoa((userID.(int)))+"/"+strconv.Itoa(f.RepositoryID)+"/"+f.Key, f.Size)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Save the file to the db.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = tx.QueryRow(ctx, "INSERT INTO file_ VALUES (DEFAULT, @repoID, @userID, @path, @type, @size, @uploadID, NULL) RETURNING id_",
		pgx.NamedArgs{"repoID": f.RepositoryID, "userID": userID, "path": f.Key, "type": "file", "size": f.Size,
			"uploadID": res.UploadID}).Scan(&res.FileID)

	// TODO: Add transaction retry.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = tx.Commit(ctx)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
