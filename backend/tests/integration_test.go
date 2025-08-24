package test

import (
	db "backend/database"
	c "backend/util/config"
	"context"
	"strconv"
	"testing"
	"time"
)

// The user that will be used for all the tests.
var testUser integrationUser

func init() {
	db.InitDB()
}

// TODO: Test deleting a user with more than 1k files (test deleteallfiles code).

// This function is going to test all main features a user may use.
func TestIntegration(t *testing.T) {
	testUser = integrationUser{
		Username: "testedUser",
		Password: "testedPassword",
	}
	// Clear the database.
	clean()

	// Test creating a user, changing his username and password, then deleting the user.
	t.Run("create a user", subtestPostUser)
	testUser.Username = "testedUser2"
	t.Run("change the username", subtestPatchUsername)
	t.Run("change the password", subtestPatchPassword)
	t.Run("delete the created user", subtestDeleteUser)

	// Test creating and deleting a user with an expired JWT, but valid refresh token.
	t.Run("create a user", subtestPostUser)
	// JWT expiry time set in seconds.
	expiryTime, err := strconv.Atoi(c.JWTExpiry)
	if err != nil {
		t.Fatal(err)
	}
	// Make a request after the access token expires.
	time.Sleep(time.Second*time.Duration(expiryTime) + time.Second)
	t.Run("delete the user with the now expired JWT", subtestDeleteUser)

	// Test creating a user, logging out and in (using lower-case username),
	// then deleting the account.
	t.Run("create a user", subtestPostUser)
	t.Run("logout", subtestPostLogout)
	// Make a request after the access token expires.
	time.Sleep(time.Second*time.Duration(expiryTime) + time.Second)
	// Make sure that the user logout deleted the session in the database.
	t.Run("fail deleting the created user", subtestDeleteUserFail)
	// Test using lower case in username for login.
	testUser.Username = "testeduser2"
	t.Run("login as the created user", subtestPostLogin)
	t.Run("delete the created user after logging in", subtestDeleteUser)

	// Test creating a user and deleting his session.
	testUser.Username = "testedUser2"
	t.Run("create a user", subtestPostUser)
	t.Run("delete user's session", subtestDeleteSession)
	t.Run("login after deleting the session", subtestPostLogin)
	t.Run("delete all user's sessions", subtestDeleteSessions)

	// Create an admin user, change the previously created user's
	// role and delete that user.
	testUser.Username = "admin"
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	t.Run("change the role of the found user", subtestPatchUserRole)
	t.Run("delete the found user", subtestDeleteUserAsAdmin)

	// Test creating a repository and uploading a file with transaction retry,
	// then deleting that file by deleting the account.
	t.Run("create a repository as an admin", subtestPostRepository)
	// Test retrying a transaction.
	t.Run("upload a file", subtestPostFile)
	t.Run("delete the created admin", subtestDeleteUser)
	testUser.Username = "guest"
	t.Run("create a user", subtestPostUser)
	t.Run("fail creating repository as a guest", subtestPostRepositoryFail)

	// Test uploading an incomplete multipart file,
	// then deleting that file by deleting the account.
	testUser.Username = "admin"
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	t.Run("create a repository as an admin", subtestPostRepository)
	// Test retrying a transaction.
	t.Run("upload a file part", subtestPostFilePart)
	t.Run("delete the created admin", subtestDeleteUser)
}

// Clear the database.
func clean() {
	// Get a connection from the database.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	conn, err := db.GetConnection(ctx)
	defer conn.Release()
	if err != nil {
		panic("Failed cleaning the database after the tests: " + err.Error())
	}

	// Delete all records from the user_ table.
	_, err = conn.Exec(ctx, "TRUNCATE user_ CASCADE")
	if err != nil {
		panic("Failed cleaning the database after the tests: " + err.Error())
	}
}
