package database

import (
	f "backend/database/functions"
	p "backend/database/procedures"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func InitDB() {
	// Set up a database connection pool.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var err error
	pool, err = pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		fmt.Println("Error creating DB connection pool")
		log.Fatalln(err)
	}

	// Create the database schema on start.
	ctx, cancel = context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	// Make sure procedures/functions are created after any table they use.
	tables := userSchema + sessionSchema + repositorySchema + fileSchema + filePartSchema + contributorSchema
	createSchema := "START TRANSACTION;" + tables + p.CreateProcedures + f.CreateFunctions + "COMMIT;"
	// Use Exec instead of Query to use multiple statements.
	_, err = pool.Exec(ctx, createSchema)
	if errors.Is(err, context.DeadlineExceeded) {
		log.Fatalln("DB not responding - Cannot create schema")
	}
	if err != nil {
		fmt.Println("Error creating DB schema")
		log.Fatal(err)
	}
}

func GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return pool.Acquire(ctx)
}
