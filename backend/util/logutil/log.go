package logutil

import (
	logdb "backend/logdatabase"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Create a log in the log database.
func Log(ip string, userID int, username string, executionTime float64, endpoint string, method string, status int) {
	// Get a connection from the log database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := logdb.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		fmt.Println("Failed to log data: " + err.Error())
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "INSERT INTO log_ VALUES (DEFAULT, CURRENT_TIMESTAMP(0), @ip, @userID, @username, @time, @endpoint, @method, @status)",
		pgx.NamedArgs{"ip": ip, "userID": userID, "username": username, "time": executionTime, "endpoint": endpoint, "method": method, "status": status})
	if err != nil {
		fmt.Println("Failed to log data: " + err.Error())
		return
	}
}

// log_ table schema:
// id_		 INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
// date_ 	 TIMESTAMPTZ NOT NULL,
// ip_ 	  	 TEXT NOT NULL CHECK (TRIM(ip_) <> ''),
// user_id_  INT NOT NULL,
// username_ TEXT,
// time_	 REAL NOT NULL,
// endpoint_ TEXT NOT NULL CHECK (TRIM(endpoint_) <> ''),
// method_	 TEXT NOT NULL CHECK (TRIM(method_) <> ''),
// status_	 INT NOT NULL
