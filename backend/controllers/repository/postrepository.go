package repository

import (
	db "backend/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type repository struct {
	Visibility string
	Name       string
}

type repositoryResponse struct {
	ID int `json:"id"`
}

func PostRepository(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("id")
	repo := repository{}
	err := json.NewDecoder(io.LimitReader(r.Body, 1000)).Decode(&repo)
	if err != nil {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}
	if repo.Visibility != "public" && repo.Visibility != "private" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(repo.Name) > 35 || utf8.RuneCountInString(repo.Name) == 0 || utf8.RuneCountInString(repo.Visibility) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get a connection from the database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create the repository.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	res := repositoryResponse{}
	err = conn.QueryRow(ctx, "CALL create_repository_(@userID, @name, @visibility, @repoID)",
		pgx.NamedArgs{"userID": userID, "name": repo.Name, "visibility": repo.Visibility, "repoID": nil}).Scan(&res.ID)
	if err != nil && strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
		fmt.Println(err)
		w.WriteHeader(http.StatusConflict)
		return
	}
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok && pgErr.Code == "01007" {
		fmt.Println(err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
