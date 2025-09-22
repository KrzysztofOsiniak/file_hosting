package file

import (
	db "backend/database"
	"backend/storage"
	"backend/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type downloadResponse struct {
	URL string `json:"url"`
}

// Get a presigned download url to an uploaded file.
// If the repository is private check if the user can download this file.
func GetDownload(w http.ResponseWriter, r *http.Request) {
	fileID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var userID int
	if r.Context().Value(types.ContextKey("id")) != nil {
		userID = r.Context().Value(types.ContextKey("id")).(int)
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

	// Get the id, visibility, owner of the repository the file is in and the file's id.
	var repositoryID int
	var visibility string
	var ownerUserID int
	var filePath string
	err = tx.QueryRow(ctx, `SELECT repository_.id_, repository_.visibility_, repository_.user_id_, file_.path_ FROM repository_ JOIN 
	file_ ON repository_.id_ = file_.repository_id_ WHERE file_.id_ = $1 LIMIT 1`, fileID).Scan(&repositoryID, &visibility, &ownerUserID, &filePath)
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

	// If the repository is private and the user is not logged in return status 401.
	if visibility == "private" && userID == 0 {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Make sure the user is the repository's member or its owner, otherwise return 403.
	if visibility == "private" && userID != ownerUserID {
		var found bool
		err = tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM member_ WHERE repository_id_ = $1 AND user_id_ = $2)",
			repositoryID, userID).Scan(&found)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Get the download url.
	url, err := storage.GetDownload(ctx, strconv.Itoa(fileID), path.Base(filePath))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res := downloadResponse{URL: url}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
