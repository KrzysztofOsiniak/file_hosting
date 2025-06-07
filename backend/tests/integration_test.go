package test

import (
	db "backend/database"
	c "backend/util/config"
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"
)

type user struct {
	Username string
	Password string
	Cookies  []*http.Cookie
}

// The user that will be used for all the tests.
var testUser user

// This function is going to test all main features a user may use.
func TestIntegration(t *testing.T) {
	testUser = user{
		Username: "testedUser",
		Password: "testedPassword",
	}
	// Clear the database after the tests to be able to run the tests again.
	defer clean()

	// Test creating and deleting a user.
	ok := t.Run("create a user", subtestPostUser)
	if !ok {
		t.Error("Failed creating a user")
		return
	}
	ok = t.Run("delete the created user", subtestDeleteUser)
	if !ok {
		t.Error("Failed deleting the created user")
		return
	}

	// Test creating and deleting a user with an expired JWT, but valid refresh token.
	ok = t.Run("create a user", subtestPostUser)
	if !ok {
		t.Error("Failed creating a user")
		return
	}
	// JWT expiry time set in seconds.
	expiryTime, err := strconv.Atoi(c.JWTExpiry)
	if err != nil {
		t.Error(err)
		return
	}
	// Make a request after the access token expires.
	time.Sleep(time.Second*time.Duration(expiryTime) + time.Second)
	ok = t.Run("delete the user with the now expired JWT", subtestDeleteUser)
	if !ok {
		t.Error("Failed deleting the created user with an expired JWT")
		return
	}

	// Test creating a user, logging out and in, then deleting the account.
	ok = t.Run("create a user", subtestPostUser)
	if !ok {
		t.Error("Failed creating a user")
		return
	}
	ok = t.Run("logout", subtestPostLogout)
	if !ok {
		t.Error("Failed logging out")
		return
	}
	// Make a request after the access token expires.
	time.Sleep(time.Second*time.Duration(expiryTime) + time.Second)
	// Make sure that the user logout deleted the session in the database.
	ok = t.Run("fail deleting the created user", subtestDeleteUserFail)
	if !ok {
		t.Error("Deleted the user account with an expired access token and deleted refresh token")
		return
	}
	ok = t.Run("login as the created user", subtestPostLogin)
	if !ok {
		t.Error("Failed logging in")
		return
	}
	ok = t.Run("delete the created user after logging in", subtestDeleteUser)
	if !ok {
		t.Error("Failed deleting the created user")
		return
	}
}

// Clear the database after running the tests.
func clean() {
	db.InitDB()
	// Get a connection from the database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		panic("Failed cleaning the database after the tests: " + err.Error())
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// Delete all records from the user_ table.
	_, err = conn.Exec(ctx, "TRUNCATE user_ CASCADE")
	if err != nil {
		panic("Failed cleaning the database after the tests: " + err.Error())
	}
}
