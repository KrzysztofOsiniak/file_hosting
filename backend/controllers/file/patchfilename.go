package file

import (
	db "backend/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fileNamePatch struct {
	Name string
	ID   int
}

func PatchFileName(w http.ResponseWriter, r *http.Request) {
	f := fileNamePatch{}
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
	userID := r.Context().Value("id").(int)

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

		// Get the file path to change.
		var filePath string
		err = tx.QueryRow(ctx, "SELECT path_ FROM file_ WHERE id_ = $1", f.ID).Scan(&filePath)
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
		var newPath string
		if path.Dir(filePath) != "." {
			newPath = path.Dir(filePath) + "/" + f.Name
		} else {
			newPath = f.Name
		}

		// Change the file path.
		_, err = tx.Exec(ctx, "UPDATE file_ SET path_ = $1 WHERE id_ = $2 AND type_ = 'file'::file_type_enum_", newPath, f.ID)
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
