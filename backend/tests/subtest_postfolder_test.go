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

type folder struct {
	Key          string
	RepositoryID int
}

type folderResponse struct {
	ID int
}

func subtestPostFolder(t *testing.T) {
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

	// Add secondTestUser to repository.
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	var folderPath string
	if testUser.FolderPath == "" {
		folderPath = "folder"
	} else {
		folderPath = "/folder"
	}
	folder := folder{Key: testUser.FolderPath + folderPath, RepositoryID: testUser.RepositoryID}
	marshalled, err := json.Marshal(folder)
	if err != nil {
		t.Fatal("Error marshalling body to be sent")
	}
	// Wrap NewReader in NopCloser to get ReadCloser.
	body := io.NopCloser(bytes.NewReader(marshalled))
	request := &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/file/folder"}, Proto: "2.0", Header: header, Body: body}
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
		t.Fatal("Server did not reply with 200 on POST folder")
	}

	folderResponse := folderResponse{}
	if err := json.NewDecoder(res.Body).Decode(&folderResponse); err != nil {
		t.Fatal("Error decoding JSON:", err)
	}
	if testUser.FolderPath == "" {
		testUser.FolderPath = "folder"
	} else {
		testUser.FolderPath += "/folder"
	}
	testUser.FolderID = folderResponse.ID
}
