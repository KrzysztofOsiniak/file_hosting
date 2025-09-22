package repository

import (
	db "backend/database"
	"backend/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type getRepositoryResponse struct {
	Name           string   `json:"name"`
	Members        []member `json:"members"`
	Files          []file   `json:"files"`
	UserPermission string   `json:"userPermission"`
}

// UploadDate is int Unix time.
type file struct {
	ID            int    `json:"id"`
	OwnerUsername string `json:"ownerUsername"`
	Path          string `json:"path"`
	Type          string `json:"type"`
	Size          int    `json:"size"`
	UploadDate    int    `json:"uploadDate"`
}

type member struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Permission string `json:"permission"`
}

// Return all data for a repository view and actions, for a given user:
//
// - User not logged in: repository's name, files and "none" UserPermission in response.
//
// - User logged in and owner: repository's name, members, files and "owner" UserPermission in response.
//
// - User logged in and member: repository's name, members (without their permission), files and "full"/"read" UserPermission in response.
//
// If the repository is not public and the user is not logged in return status 401.
// If the user is logged in, the repository is private, the user is not the repository's owner and not its member return 403.
// For each file and member add the username of the file's or member's user_id_.
func GetRepository(w http.ResponseWriter, r *http.Request) {
	repositoryID, err := strconv.Atoi(chi.URLParam(r, "id"))
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

	// Get the name, visibility and owner of the repository.
	var res getRepositoryResponse
	var visibility string
	var ownerUserID int
	err = tx.QueryRow(ctx, "SELECT name_, visibility_, user_id_ FROM repository_ WHERE id_ = $1",
		repositoryID).Scan(&res.Name, &visibility, &ownerUserID)
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
	if userID == ownerUserID {
		res.UserPermission = "owner"
	} else {
		res.UserPermission = "none"
	}

	// If the repository is private and the user is not logged in return status 401.
	if visibility == "private" && userID == 0 {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// If the user is logged in and is not the repository's owner, check if they are a member.
	// Add the member's permission to the response.
	if userID != 0 && userID != ownerUserID {
		err = tx.QueryRow(ctx, "SELECT permission_ FROM member_ WHERE repository_id_ = $1 AND user_id_ = $2",
			repositoryID, userID).Scan(&res.UserPermission)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// If the user is logged in, the repository is private, the user is not the repository's owner and not its member return 403.
	var isMember bool
	if res.UserPermission == "full" || res.UserPermission == "read" {
		isMember = true
	}
	if userID != 0 && visibility == "private" && userID != ownerUserID && !isMember {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Get all files with usernames in the repository.
	rows, err := tx.Query(ctx, `SELECT file_.id_, user_.username_, file_.path_, file_.type_, file_.size_, file_.upload_date_
		FROM file_ JOIN user_ ON file_.user_id_ = user_.id_ WHERE file_.repository_id_ = $1`, repositoryID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Files, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (file, error) {
		var file file
		var date *time.Time
		err := row.Scan(&file.ID, &file.OwnerUsername, &file.Path, &file.Type, &file.Size, &date)
		if date == nil {
			file.UploadDate = 0
		} else {
			file.UploadDate = int(date.Unix())
		}
		return file, err
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// If the user is not logged in, send the response.
	if userID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
		return
	}

	// Get the repository's members with their usernames.
	rows, err = tx.Query(ctx, `SELECT member_.id_, user_.username_, member_.permission_
		FROM member_ JOIN user_ ON member_.user_id_ = user_.id_ WHERE member_.repository_id_ = $1`, repositoryID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Members, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (member, error) {
		var member member
		err := row.Scan(&member.ID, &member.Username, &member.Permission)
		// If the user making the request is a member, permissions in members should be empty.
		if isMember {
			member.Permission = ""
		}
		return member, err
	})
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
