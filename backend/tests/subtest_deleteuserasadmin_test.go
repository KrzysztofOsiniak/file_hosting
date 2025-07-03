package test

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

// Test deleting a user as an admin, after getting a list of users.
func subtestDeleteUserAsAdmin(t *testing.T) {
	// Get a new SystemCertPool.
	rootCAs, err := loadCerts()
	if err != nil {
		t.Error(err)
		return
	}

	// Trust the augmented cert pool in our client.
	config := &tls.Config{
		InsecureSkipVerify: false,
		RootCAs:            rootCAs,
	}
	tr := &http.Transport{TLSClientConfig: config, ForceAttemptHTTP2: true}
	client := &http.Client{Transport: tr}

	// Get users with "user" in their username.
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	request := &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/admin/users/user"}, Proto: "2.0", Header: header}
	if len(testUser.Cookies) == 0 {
		t.Error("Found no user's cookies to be sent")
		return
	}
	request.AddCookie(testUser.Cookies[0])
	res, err := client.Do(request)
	if err != nil || res == nil {
		t.Error("Server request error")
		return
	}
	if res.StatusCode != 200 {
		t.Error("Server did not reply with 200 on GET users")
		return
	}
	defer res.Body.Close()
	var users allUsers
	if err := json.NewDecoder(res.Body).Decode(&users); err != nil {
		t.Error("Error decoding JSON:", err)
		return
	}
	if len(users.Users) == 0 {
		t.Error("Server returned an empty user array")
		return
	}

	// Delete one of the users from the get request.
	request = &http.Request{Method: "DELETE", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/admin/user/" + strconv.Itoa(users.Users[0].ID)}, Proto: "2.0", Header: header}
	if len(testUser.Cookies) == 0 {
		t.Error("Found no user's cookies to be sent")
		return
	}
	request.AddCookie(testUser.Cookies[0])
	res, err = client.Do(request)
	if err != nil || res == nil {
		t.Error("Server request error")
		return
	}
	if res.StatusCode != 200 {
		t.Error("Server did not reply with 200 on GET users")
		return
	}
}
