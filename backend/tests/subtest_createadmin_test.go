package test

import (
	db "backend/database"
	"context"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
)

// Test creating an admin with a direct call to the db.
func subtestCreateAdmin(t *testing.T) {
	// Hash is salted by default
	hash, err := argon2id.CreateHash(testUser.Password, argon2id.DefaultParams)
	if err != nil {
		t.Error(err)
		return
	}

	// Get a connection from the database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		t.Error(err)
		return
	}

	// Create the user in the database.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = conn.Exec(ctx, "INSERT INTO user_ VALUES (DEFAULT, $1, $2, 'admin')", testUser.Username, hash)
	if err != nil {
		t.Error(err)
		return
	}
}
