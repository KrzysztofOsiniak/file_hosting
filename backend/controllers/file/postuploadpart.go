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
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type filePartRequest struct {
	types.CompletePart
	FileID int
}

func PostUploadPart(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("id")
	part := filePartRequest{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&part)
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

	// Insert the file part.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "CALL create_file_part_(@fileID, @eTag, @part, @userID)",
		pgx.NamedArgs{"fileID": part.FileID, "eTag": part.ETag, "part": part.Part, "userID": userID})
	if err != nil && strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
		fmt.Println(err)
		w.WriteHeader(http.StatusConflict)
		return
	}
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
	w.WriteHeader(http.StatusOK)
}
