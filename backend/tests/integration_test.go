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

// The user that testUser can use, for example to add as a repository member.
var secondTestUser integrationUser

func init() {
	db.InitDB()
}

// TODO: Test deleting a user with more than 1k files (test deleteallfiles code).
// TODO: Test uploading a file with not enough space in user's account.

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
	t.Run("upload a file part", subtestPostFilePart)
	t.Run("delete the created admin", subtestDeleteUser)

	// Test adding a member and uploading a file in a folder to a repository as that member.
	// Create an admin, then create a second admin that creates a repository and adds the first admin to it.
	// The first admin then uploads a file to that repository as its member.
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	secondTestUser = testUser
	testUser.Username = "admin2"
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	t.Run("create a repository as an admin", subtestPostRepository)
	tempUser := testUser
	t.Run("add secondTestUser to testUser's repository", subtestPostMember)
	testUser = secondTestUser
	testUser.RepositoryID = tempUser.RepositoryID
	t.Run("create a folder", subtestPostFolder)
	t.Run("upload a file", subtestPostFile)
	t.Run("delete the account along with its uploads", subtestDeleteUser)
	testUser = tempUser
	t.Run("delete the first admin", subtestDeleteUser)
	testUser.FolderPath = ""

	// Test resuming an upload.
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	t.Run("create a repository as an admin", subtestPostRepository)
	t.Run("upload a file after resuming the upload", subtestResumeUpload)
	t.Run("delete the account along with its uploads", subtestDeleteUser)

	// Test aborting an upload and deleting an uploaded file.
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	t.Run("create a repository as an admin", subtestPostRepository)
	t.Run("upload a file part", subtestPostFilePart)
	t.Run("abort the upload", subtestDeleteAbortUpload)
	t.Run("upload a file", subtestPostFile)
	t.Run("delete the file", subtestDeleteFile)

	// Test removing a folder file and all other user's files in it.
	// Create 2 folders: folder/ and folder/folder/ and upload a file in folder/ and add a second user as the repository's member,
	// and upload an in progress file in folder/folder/ as that user and delete folder/.
	t.Run("create folder folder/", subtestPostFolder)
	folderID := testUser.FolderID
	t.Run("upload a file", subtestPostFile)
	t.Run("create folder folder/folder/", subtestPostFolder)
	tempUser = testUser
	testUser.Username = "admin3"
	t.Run("create an admin user", subtestCreateAdmin)
	t.Run("login as created admin", subtestPostLogin)
	secondTestUser = testUser
	testUser = tempUser
	t.Run("add secondTestUser to testUser's repository", subtestPostMember)
	testUser = secondTestUser
	t.Run("upload a file", subtestPostFilePart)
	testUser.FolderID = folderID
	t.Run("delete folder/", subtestDeleteFolder)
	testUser.FolderPath = ""

	// Test deleting a member as the repository owner, and leaving the repository's members as the added member.
	testUser = tempUser
	t.Run("delete secondTestUser from members", subtestDeleteMember)
	t.Run("add secondTestUser to testUser's repository", subtestPostMember)
	testUser = secondTestUser
	t.Run("delete secondTestUser from members as secondTestUser", subtestDeleteMember)

	// Test changing member's permission and user's storage space.
	testUser = tempUser
	secondTestUser.Username = "admin3"
	t.Run("add secondTestUser to testUser's repository", subtestPostMember)
	t.Run("change secondTestUser's permission", subtestPatchMemberPermission)
	t.Run("change secondTestUser's storage space", subtestPatchUserStorageSpace)

	// Test getting the private repository as the owner and as a member, change the repository's visibility (to public) and name,
	// then test getting the repository as a not logged in user and a logged in user that is not the owner nor a member.
	testUser = tempUser
	secondTestUser.RepositoryID = testUser.RepositoryID
	t.Run("get the repository as the owner", subtestGetRepository)
	testUser = secondTestUser
	t.Run("get the repository as a member", subtestGetRepository)
	testUser = tempUser
	t.Run("patch repository visibility", subtestPatchRepositoryVisibility)
	t.Run("patch repository name", subtestPatchRepositoryName)
	// Test logged in/out
	testUser.Username = "testUser"
	t.Run("create a user", subtestPostUser)
	t.Run("get the public repository while logged in", subtestGetRepository)
	t.Run("log out", subtestPostLogout)
	t.Run("get the repository while logged out", subtestGetRepository)
	testUser = tempUser

	// Upload a file and change its name, then upload a file into a folder and change the files name.
	testUser.FolderPath = ""
	t.Run("upload a file", subtestPostFile)
	t.Run("change the file name", subtestPatchFileName)
	t.Run("create a folder", subtestPostFolder)
	t.Run("upload a file", subtestPostFile)
	t.Run("change the file name", subtestPatchFileName)

	// Test deleting user's repository.
	t.Run("delete user's repository", subtestDeleteRepository)

	// Test uploading 2 files to a folder, then changing that folder's name.
	testUser.FolderPath = ""
	t.Run("create a repository as an admin", subtestPostRepository)
	t.Run("create a folder", subtestPostFolder)
	t.Run("upload a file", subtestPostFile)
	t.Run("change the file name", subtestPatchFileName)
	t.Run("upload a file", subtestPostFile)
	t.Run("change folder's file name", subtestPatchFolderName)
	t.Run("delete user's repository", subtestDeleteRepository)
	testUser.FolderPath = ""
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
