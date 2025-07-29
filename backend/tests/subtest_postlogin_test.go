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

func subtestPostLogin(t *testing.T) {
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
	user := integrationUser{
		Username: testUser.Username,
		Password: testUser.Password,
	}
	marshalled, err := json.Marshal(user)
	if err != nil {
		t.Fatal("Error marshalling body to be sent")
	}
	// Wrap NewReader in NopCloser to get ReadCloser.
	body := io.NopCloser(bytes.NewReader(marshalled))
	res, err := client.Do(&http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/user/login"}, Proto: "2.0", Header: header, Body: body})
	if err != nil || res == nil {
		t.Fatal("Server request error")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("Server did not reply with 200 on POST login")
	}
	testUser.Cookies = res.Cookies()
}
