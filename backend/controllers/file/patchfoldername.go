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
	"path"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type folderNamePatch struct {
	Name string
	ID   int
}

type fileChangePath struct {
	ID   int
	Path string
}

// Change a folder's name and path, and the path of all other files containing the folder's path.
func PatchFolderName(w http.ResponseWriter, r *http.Request) {
	f := folderNamePatch{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&f)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	f.Name = path.Clean(f.Name)
	// Make sure the file name is not an empty string.
	if f.Name == "." {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Check if the file name is valid.
	runes := []rune(f.Name)
	if string(runes[0]) == "/" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Make sure the file name is not actually a path.
	if path.Dir(f.Name) != "." {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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

	// Check if the user can modify this file.
	_, err = tx.Exec(ctx, "CALL check_permission_modify_file_(@userID, @fileID)", pgx.NamedArgs{"userID": userID, "fileID": f.ID})
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
	err = tx.Commit(ctx)
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

		// Get the folder's file path to change and the repository id to get more files later.
		var folderPath string
		var repositoryID int
		err = tx.QueryRow(ctx, "SELECT path_, repository_id_ FROM file_ WHERE id_ = $1", f.ID).Scan(&folderPath, &repositoryID)
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

		// Create a new file path from the directory of the file and the new file name, or if the path has no folders then from the name alone.
		var newFolderPath string
		if path.Dir(folderPath) != "." {
			newFolderPath = path.Dir(folderPath) + "/" + f.Name
		} else {
			newFolderPath = f.Name
		}

		// Change the file path of the folder.
		_, err = tx.Exec(ctx, "UPDATE file_ SET path_ = $1 WHERE id_ = $2 AND type_ = 'folder'::file_type_enum_", newFolderPath, f.ID)
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

		// Get id_ and path_ of all files inside the folder.
		rows, err := tx.Query(ctx, "SELECT id_, path_ FROM file_ WHERE repository_id_ = $1 AND path_ LIKE $2 || '/%'", repositoryID, folderPath)
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
		files, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (fileChangePath, error) {
			var file fileChangePath
			err := row.Scan(&file.ID, &file.Path)
			return file, err
		})
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Update each file with a new path in the db.
		for _, file := range files {
			_, err = tx.Exec(ctx, "UPDATE file_ SET path_ = $1 WHERE id_ = $2", strings.Replace(file.Path, folderPath, newFolderPath, 1), file.ID)
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
