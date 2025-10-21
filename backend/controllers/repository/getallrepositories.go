package repository

import (
	db "backend/database"
	"backend/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type getAllRepositoriesRes struct {
	Repositories []repositoryData `json:"repositories"`
}
type repositoryData struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	OwnerUsername     string `json:"ownerUsername"`
	UserUploadedSpace int    `json:"userUploadedSpace"`
}

// Get all repositories the user owns or is a member of.
func GetAllRepositories(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(types.ContextKey("id"))
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

	// Get owned repositories.
	var res getAllRepositoriesRes
	rows, err := tx.Query(ctx, `SELECT r.id_, r.name_, u.username_, COALESCE(SUM(f.size_), 0) FROM repository_ r 
	JOIN user_ u ON r.user_id_ = u.id_ JOIN file_ f ON r.id_ = f.repository_id_ AND r.user_id_ = f.user_id_ 
	WHERE r.user_id_ = @userID GROUP BY r.id_, r.name_, u.username_`, pgx.NamedArgs{"userID": userID})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Repositories, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (repositoryData, error) {
		var repo repositoryData
		err := row.Scan(&repo.ID, &repo.Name, &repo.OwnerUsername, &repo.UserUploadedSpace)
		return repo, err
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get repositories where the user is a member.
	rows, err = tx.Query(ctx, `SELECT r.id_, r.name_, u.username_, COALESCE(SUM(f.size_), 0) FROM repository_ r 
	JOIN user_ u ON r.user_id_ = u.id_ JOIN member_ m ON r.id_ = m.repository_id_ JOIN file_ f ON r.id_ = f.repository_id_
	WHERE m.user_id_ = @userID AND f.user_id_ = @userID GROUP BY r.id_, r.name_, u.username_`,
		pgx.NamedArgs{"userID": userID})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	memberRepos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (repositoryData, error) {
		var repo repositoryData
		err := row.Scan(&repo.ID, &repo.Name, &repo.OwnerUsername, &repo.UserUploadedSpace)
		return repo, err
	})
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Repositories = append(res.Repositories, memberRepos...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
