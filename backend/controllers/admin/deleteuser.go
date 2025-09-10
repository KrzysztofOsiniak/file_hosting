package admin

import (
	db "backend/database"
	"backend/storage"
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

type allFiles struct {
	Files []fileData
}

type fileData struct {
	UserID       int
	RepositoryID int
	Path         string
	Date         *time.Time
	UploadID     string
}

// Delete a user as an admin and delete files in their repositories.
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	deleteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		fmt.Println(err)
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

	// Get all files in the user's repositories and delete them in a loop later.
	rows, err := tx.Query(ctx, `SELECT file_.user_id_, file_.repository_id_, file_.path_, file_.upload_date_, file_.upload_id_ FROM file_ JOIN repository_ ON file_.repository_id_ = repository_.id_
		WHERE repository_.user_id_ = @userID AND file_.type_ = 'file'::file_type_enum_`,
		pgx.NamedArgs{"userID": deleteID})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Scan the rows into an array.
	files := allFiles{}
	for rows.Next() {
		file := fileData{}
		err = rows.Scan(&file.UserID, &file.RepositoryID, &file.Path, &file.Date, &file.UploadID)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		files.Files = append(files.Files, file)
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
	for _, file := range files.Files {
		if file.Date == nil {
			err = storage.AbortUpload(ctx, strconv.Itoa(file.UserID)+"/"+strconv.Itoa(file.RepositoryID)+"/"+file.Path, file.UploadID)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			continue
		}
		err = storage.DeleteFile(ctx, strconv.Itoa(file.UserID)+"/"+strconv.Itoa(file.RepositoryID)+"/"+file.Path)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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
		_, err = tx.Exec(ctx, "DELETE FROM file_ WHERE user_id_ = $1 AND type_ = 'file'::file_type_enum_", deleteID)
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
		_, err = tx.Exec(ctx, "DELETE FROM user_ WHERE id_ = $1", deleteID)
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
