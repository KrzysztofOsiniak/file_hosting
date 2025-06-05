package test

import (
	c "backend/util/config"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
)

// Test sending a body that's > 1kB.
func TestBodyTooLarge(t *testing.T) {
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

	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	user := user{
		Username: "bodytest",
		Password: "password",
	}
	for i := 1; i < 400; i++ {
		user.Username += "Lorem ipsum dolor sit amet, consectetur adipiscing elit"
	}
	marshalled, err := json.Marshal(user)
	if err != nil {
		t.Error("Error marshalling body to be sent")
		return
	}
	// Wrap NewReader in NopCloser to get ReadCloser.
	body := io.NopCloser(bytes.NewReader(marshalled))
	res, _ := client.Do(&http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: c.ServerHost, Path: "/user/"}, Proto: "2.0", Header: header, Body: body})
	if res.StatusCode != 400 {
		t.Error("Server did not refuse a body that's too large")
	}
}
