package test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// Test sending a header that's > 8kB.
func TestHeaderTooLarge(t *testing.T) {
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
	for i := 1; i < 400; i++ {
		header.Set(strconv.Itoa(i), "Lorem ipsum dolor sit amet, consectetur adipiscing elit")
	}
	user := integrationUser{
		Username: "headertest",
		Password: "password",
	}
	marshalled, err := json.Marshal(user)
	if err != nil {
		t.Fatal("Error marshalling body to be sent")
	}
	// Wrap NewReader in NopCloser to get ReadCloser.
	body := io.NopCloser(bytes.NewReader(marshalled))
	_, err = client.Do(&http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/user/"}, Proto: "2.0", Header: header, Body: body})
	if err == nil || !strings.Contains(err.Error(), "GOAWAY") {
		t.Fatal("Server did not refuse a header that's too large")
	}
}
