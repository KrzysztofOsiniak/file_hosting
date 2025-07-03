package test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

// Search for users with "user" in their username, and change one's role.
func subtestPatchUserRole(t *testing.T) {
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

	// Patch a user's role.
	user := user{
		Role: "user",
	}
	marshalled, err := json.Marshal(user)
	if err != nil {
		t.Error("Error marshalling body to be sent")
		return
	}
	// Wrap NewReader in NopCloser to get ReadCloser.
	body := io.NopCloser(bytes.NewReader(marshalled))
	request = &http.Request{Method: "PATCH", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/admin/user/role/" + strconv.Itoa(users.Users[0].ID)}, Proto: "2.0", Header: header, Body: body}
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
		t.Error("Server did not reply with 200 on PATCH user/role")
		return
	}
}
