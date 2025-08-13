package file

import (
	db "backend/database"
	"backend/storage"
	"backend/types"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type uploadCompleteRequest struct {
	FileKey      string
	UploadID     string
	FileID       int
	RepositoryID int
}

func PostUploadComplete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("id").(int)
	req := uploadCompleteRequest{}
	err := json.NewDecoder(io.LimitReader(r.Body, 10*1000)).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
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

	// Get the parts.
	rows, err := tx.Query(ctx, "SELECT * FROM get_file_parts_(@fileID, @userID)",
		pgx.NamedArgs{"fileID": req.FileID, "userID": userID})
	if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "01007" {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Scan the rows into an array.
	parts := []types.CompletePart{}
	for rows.Next() {
		part := types.CompletePart{}
		err = rows.Scan(&part.ETag, &part.Part)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		parts = append(parts, part)
	}
	if rows.Err() != nil {
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

	err = storage.CompleteUpload(strconv.Itoa(userID)+"/"+strconv.Itoa(req.RepositoryID)+"/"+req.FileKey, req.UploadID, parts)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: Update the file's date in DB.

	w.WriteHeader(http.StatusOK)
}
