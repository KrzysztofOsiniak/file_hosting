package test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
)

type userStorage struct {
	ID     int
	Amount int // The amount in bytes.
}

// Test deleting a user.
func subtestPatchUserStorageSpace(t *testing.T) {
	// Get a new SystemCertPool.
	rootCAs, err := loadCerts()
	if err != nil {
		t.Fatal(err)
	}

	// Trust the augmented cert pool in our client.
	config := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            rootCAs,
	}
	tr := &http.Transport{TLSClientConfig: config, ForceAttemptHTTP2: true}
	client := &http.Client{Transport: tr}

	// Get users with "admin3" in their username.
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	request := &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/admin/users/admin3"}, Proto: "2.0", Header: header}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	request.AddCookie(testUser.Cookies[0])
	res, err := client.Do(request)
	if err != nil || res == nil {
		t.Fatal("Server request error")
	}
	if res.StatusCode != 200 {
		t.Fatal("Server did not reply with 200 on GET users")
	}
	defer res.Body.Close()
	var users allUsers
	if err := json.NewDecoder(res.Body).Decode(&users); err != nil {
		t.Fatal("Error decoding JSON:", err)
	}
	if len(users.Users) == 0 {
		t.Fatal("Server returned an empty user array")
	}

	// Add secondTestUser to repository.
	header = http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	user := userStorage{ID: users.Users[0].ID, Amount: 1000000}
	marshalled, err := json.Marshal(user)
	if err != nil {
		t.Fatal("Error marshalling body to be sent")
	}
	// Wrap NewReader in NopCloser to get ReadCloser.
	body := io.NopCloser(bytes.NewReader(marshalled))
	request = &http.Request{Method: "PATCH", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/admin/user/storage-space"}, Proto: "2.0", Header: header, Body: body}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	request.AddCookie(testUser.Cookies[0])
	res, err = client.Do(request)
	if err != nil || res == nil {
		t.Fatal("Server request error")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("Server did not reply with 200 on PATCH user storage space")
	}
}
