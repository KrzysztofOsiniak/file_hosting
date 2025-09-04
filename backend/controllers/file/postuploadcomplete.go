package file

import (
	db "backend/database"
	"backend/storage"
	"backend/types"
	"backend/util/fileutil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type uploadComplete struct {
	FileID int
}

func PostUploadComplete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("id").(int)
	req := uploadComplete{}
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

	// Get the parts and check if user owns this file.
	rows, err := tx.Query(ctx, "SELECT * FROM get_file_parts_(@fileID, @userID)",
		pgx.NamedArgs{"fileID": req.FileID, "userID": userID})
	var pgErr *pgconn.PgError
	ok := errors.As(err, &pgErr)
	if ok && pgErr.Code == pgerrcode.PrivilegeNotGranted {
		fmt.Println(err)
		w.WriteHeader(http.StatusForbidden)
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
	if len(parts) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the file size to know how many parts are required and check if that amount matches.
	var size int
	err = tx.QueryRow(ctx, "SELECT size_ FROM file_ WHERE id_ = $1", req.FileID).Scan(&size)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	partCount, _, _ := fileutil.SplitFile(size)
	if partCount != len(parts) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the file to complete upload for.
	var (
		path         string
		uploadID     string
		repositoryID int
	)
	err = tx.QueryRow(ctx, "SELECT path_, upload_id_, repository_id_ FROM file_ WHERE id_ = @fileID AND user_id_ = @userID AND type_ = 'file'::file_type_enum_",
		pgx.NamedArgs{"fileID": req.FileID, "userID": userID}).Scan(&path, &uploadID, &repositoryID)
	if errors.Is(err, pgx.ErrNoRows) {
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

	err = storage.CompleteUpload(ctx, strconv.Itoa(userID)+"/"+strconv.Itoa(repositoryID)+"/"+path, uploadID, parts)
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

		// Update the file's date in db from null, to mark the file has been fully uploaded.
		_, err = tx.Exec(ctx, "UPDATE file_ SET upload_date_ = CURRENT_TIMESTAMP(0) WHERE id_ = $1", req.FileID)
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

	w.WriteHeader(http.StatusOK)
}
