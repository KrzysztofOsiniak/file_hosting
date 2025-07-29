package test

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"
)

// Test deleting a user.
func subtestPostLogout(t *testing.T) {
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

	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	request := &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/user/logout"}, Proto: "2.0", Header: header}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	request.AddCookie(testUser.Cookies[0])
	res, err := client.Do(request)
	if err != nil || res == nil {
		t.Fatal("Server request error")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("Server did not reply with 200 on POST logout")
	}
}
