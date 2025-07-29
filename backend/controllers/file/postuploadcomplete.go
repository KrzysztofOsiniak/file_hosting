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

	// Get a connection from the database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get the parts.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := conn.Query(ctx, "SELECT * FROM get_file_parts_(@fileID, @userID)",
		pgx.NamedArgs{"fileID": req.FileID, "userID": userID})
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok && pgErr.Code == "01007" {
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

	err = storage.CompleteUpload(strconv.Itoa(userID)+"/"+strconv.Itoa(req.RepositoryID)+"/"+req.FileKey, req.UploadID, parts)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
