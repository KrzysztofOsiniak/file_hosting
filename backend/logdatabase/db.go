package logdatabase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Export Pool to be able to check if it's not nil for optional logging.
var Pool *pgxpool.Pool

func InitDB() {
	// Set up a database connection pool.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var err error
	Pool, err = pgxpool.New(ctx, os.Getenv("LOG_DB_URL"))
	if err != nil {
		fmt.Println("Error creating log DB connection pool")
		log.Fatalln(err)
	}

	// Create the database schema on start.
	ctx, cancel = context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	// Make sure procedures are created after any table they use.
	createSchema := "START TRANSACTION;" + logSchema + "COMMIT;"
	// Use Exec instead of Query to use multiple statements.
	_, err = Pool.Exec(ctx, createSchema)
	if errors.Is(err, context.DeadlineExceeded) {
		log.Fatalln("Log DB not responding - Cannot create schema")
	}
	if err != nil {
		fmt.Println("Error creating log DB schema")
		log.Fatal(err)
	}
}

func GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return Pool.Acquire(ctx)
}
