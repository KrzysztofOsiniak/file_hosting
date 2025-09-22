package member

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

// Delete a repository's member as the repository owner or the member himself, then delete that member's files (without folders).
func DeleteMember(w http.ResponseWriter, r *http.Request) {
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

	// Check if the user can delete this member.
	_, err = tx.Exec(ctx, "CALL check_permission_delete_member_(@userID, @memberID)", pgx.NamedArgs{"userID": userID, "memberID": id})
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

	// Get the data to remove the member's files from s3.
	var (
		memberUserID int
		repositoryID int
	)
	err = tx.QueryRow(ctx, "SELECT user_id_, repository_id_ FROM member_ WHERE id_ = $1", id).Scan(&memberUserID, &repositoryID)
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

	// Get all the member's files in the repository the member is being deleted from.
	rows, err := tx.Query(ctx, `SELECT id_, upload_date_, upload_id_ FROM file_ WHERE repository_id_ = @repositoryID AND user_id_ = @userID`,
		pgx.NamedArgs{"userID": memberUserID, "repositoryID": repositoryID})
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

		// Delete the member's files (without folders).
		_, err = tx.Exec(ctx, "DELETE FROM file_ WHERE user_id_ = $1 AND type_ = 'file'::file_type_enum_", memberUserID)
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

		// Delete the member.
		_, err = tx.Exec(ctx, "DELETE FROM member_ WHERE id_ = $1", id)
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

	w.WriteHeader(http.StatusOK)
}
