package user

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

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Called if a user tries to delete his account.
// Delete the user account and all files in user's repositories.
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get the userID from the auth middleware.
	userID := r.Context().Value(types.ContextKey("id")).(int)

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

	// Get all files the user has and all files in the user's repositories to delete them from s3.
	rows, err := tx.Query(ctx, `SELECT file_.id_, file_.upload_date_, file_.upload_id_ FROM file_ JOIN repository_ ON file_.repository_id_ = repository_.id_
		WHERE (repository_.user_id_ = @userID OR file_.user_id_ = @userID) AND file_.type_ = 'file'::file_type_enum_`, pgx.NamedArgs{"userID": userID})
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

		// Delete the user's files (not folders) that are not ON DELETE CASCADE from the database.
		_, err = tx.Exec(ctx, "DELETE FROM file_ WHERE user_id_ = $1 AND type_ = 'file'::file_type_enum_", userID)
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
		// Delete the user from the database.
		_, err = tx.Exec(ctx, "DELETE FROM user_ WHERE id_ = $1", userID)
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

	// Create an empty cookie to unset the current one.
	cookie := http.Cookie{
		Name:     "file_hosting",
		Value:    "",
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
}
