package file

import (
	db "backend/database"
	"backend/storage"
	"backend/types"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func DeleteFolder(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(types.ContextKey("id")).(int)
	idString := chi.URLParam(r, "id")
	// Check if the id to delete is a number.
	id, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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

	// Check if the user can delete this file.
	_, err = tx.Exec(ctx, "CALL check_permission_modify_file_(@userID, @fileID)", pgx.NamedArgs{"userID": userID, "fileID": id})
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

	// Get data needed for later.
	var (
		repositoryID int
		folderPath   string
	)
	err = tx.QueryRow(ctx, `SELECT repository_id_, path_ FROM file_ WHERE id_ = @fileID`,
		pgx.NamedArgs{"userID": userID, "fileID": id}).Scan(&repositoryID, &folderPath)
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

	// Delete all files in this folder.
	// Get all files with this folder's path_ at the start of their own path_ and delete them from s3.
	rows, err := tx.Query(ctx, "SELECT id_, upload_date_, upload_id_ FROM file_ WHERE repository_id_ = @repositoryID AND type_ = 'file'::file_type_enum_ AND path_ LIKE @path || '%'",
		pgx.NamedArgs{"repositoryID": repositoryID, "path": folderPath})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Scan the rows into two arrays for deletion.
	uploadedFiles := []types.UploadedFile{}
	inProgressFiles := []types.InProgressFile{}
	for rows.Next() {
		file := types.FileData{}
		err = rows.Scan(&file.ID, &file.Date, &file.UploadID)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if file.Date == nil {
			inProgressFiles = append(inProgressFiles, types.InProgressFile{ID: strconv.Itoa(file.ID), UploadID: file.UploadID})
		} else {
			uploadedFiles = append(uploadedFiles, types.UploadedFile{ID: strconv.Itoa(file.ID)})
		}
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
	// Delete files from s3.
	err = storage.DeleteAllFiles(ctx, uploadedFiles, inProgressFiles)
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

		// Delete the files.
		_, err = tx.Exec(ctx, "DELETE FROM file_ WHERE repository_id_ = @repositoryID AND (path_ LIKE @path || '/%' OR path_ = @path)",
			pgx.NamedArgs{"repositoryID": repositoryID, "path": folderPath})
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
