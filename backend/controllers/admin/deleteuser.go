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

// Delete a user as an admin.
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

	// Check if user has any (fully) uploaded files and delete them.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly, DeferrableMode: pgx.Deferrable})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// If commit is not run first this will rollback the transaction.
	defer tx.Rollback(ctx)
	var exists bool
	err = tx.QueryRow(ctx, "SELECT 1 FROM file_ WHERE user_id_ = $1 AND upload_date_ <> NULL", deleteID).Scan(&exists)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	txErr := tx.Commit(ctx)
	if txErr != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// If user has files uploaded.
	if err == nil {
		// Delete user's uploaded files.
		err = storage.DeleteAllFiles(ctx, strconv.Itoa(deleteID))
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

		// Delete the user from the database.
		_, err = tx.Exec(ctx, "DELETE from user_ WHERE id_ = $1", deleteID)
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
