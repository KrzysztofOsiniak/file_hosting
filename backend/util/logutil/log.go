package logutil

import (
	logdb "backend/logdatabase"
	"context"
	"fmt"
	"time"
)

// Log a successful request into the log database.
// This logging is used for security reasons since users can post arbitrary content.
func Log(ip string, userID int, username string, endpoint string, method string) {
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
	_, err = conn.Exec(ctx, "INSERT INTO log_ VALUES (DEFAULT, CURRENT_TIMESTAMP(0), $1, $2, $3, $4, $5)", ip, userID, username, endpoint, method)
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
// endpoint_ TEXT NOT NULL CHECK (TRIM(endpoint_) <> ''),
// method_	 TEXT NOT NULL CHECK (TRIM(method_) <> '')
