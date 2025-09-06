package file

import (
	db "backend/database"
	"backend/storage"
	"backend/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type resumeFile struct {
	ID int
}

type resumeFileResponse struct {
	UploadParts []types.UploadPart `json:"uploadParts"`
}

func PostResumeUpload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("id").(int)
	f := resumeFile{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&f)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
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
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly, DeferrableMode: pgx.Deferrable})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// If commit is not run first this will rollback the transaction.
	defer tx.Rollback(ctx)

	// Get the file to resume upload for.
	var (
		path         string
		uploadID     string
		bytes        int
		repositoryID int
	)
	err = tx.QueryRow(ctx, "SELECT path_, upload_id_, size_, repository_id_ FROM file_ WHERE id_ = @fileID AND user_id_ = @userID AND type_ = 'file'::file_type_enum_",
		pgx.NamedArgs{"fileID": f.ID, "userID": userID}).Scan(&path, &uploadID, &bytes, &repositoryID)
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

	// Get the file's uploaded parts.
	rows, err := tx.Query(ctx, "SELECT part_ FROM file_part_ WHERE file_id_ = $1", f.ID)
	// Scan the rows into an array.
	completeParts := []types.CompletePart{}
	for rows.Next() {
		part := types.CompletePart{}
		err = rows.Scan(&part.Part)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		completeParts = append(completeParts, part)
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

	// Get presigned urls for uploads.
	res := resumeFileResponse{}
	res.UploadParts, err = storage.ResumeUpload(ctx, strconv.Itoa(userID)+"/"+strconv.Itoa(repositoryID)+"/"+path, uploadID, bytes, completeParts)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
