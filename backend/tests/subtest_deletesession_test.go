package test

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

type allSessions struct {
	Sessions []session `json:"sessions"`
}

type session struct {
	ID         int    `json:"id"`
	ExpiryDate string `json:"expirydate"`
	Device     string `json:"device"`
}

// Get all user's sessions and delete one.
func subtestDeleteSession(t *testing.T) {
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

	// Get all user's sessions.
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	request := &http.Request{Method: "GET", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/session/all"}, Proto: "2.0", Header: header}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	request.AddCookie(testUser.Cookies[0])
	res, err := client.Do(request)
	if err != nil || res == nil {
		t.Fatal("Server request error")
	}
	defer res.Body.Close()
	var sessions allSessions
	if err := json.NewDecoder(res.Body).Decode(&sessions); err != nil {
		t.Fatal("Error decoding JSON:", err)
	}
	if len(sessions.Sessions) == 0 {
		t.Fatal("Server returned an empty session array")
	}

	// Delete a single user's session.
	request = &http.Request{Method: "DELETE", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/session/" + strconv.Itoa(sessions.Sessions[0].ID)}, Proto: "2.0", Header: header}
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
		t.Fatal("Server did not reply with 200 on DELETE session")
	}
}

// Get all user's sessions and delete one.
func subtestDeleteSessions(t *testing.T) {
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

	// Delete all user's sessions.
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	request := &http.Request{Method: "DELETE", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/session/all"}, Proto: "2.0", Header: header}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	request.AddCookie(testUser.Cookies[0])
	res, err := client.Do(request)
	if err != nil || res == nil {
		t.Fatal("Server request error")
	}
	if res.StatusCode != 200 {
		t.Fatal("Server did not reply with 200 on DELETE session")
	}
}
