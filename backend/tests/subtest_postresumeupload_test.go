package test

import (
	"backend/types"
	"backend/util/fileutil"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
)

type resumeFile struct {
	ID int
}

type resumeFileResponse struct {
	UploadParts []types.UploadPart `json:"uploadParts"`
}

// Start a multipart upload and upload the parts after resuming the upload.
func subtestResumeUpload(t *testing.T) {
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

	file, err := os.Open("integration_test.go")
	if err != nil {
		t.Fatal("failed to open file:", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatal("failed getting file information", err)
	}
	var folder string
	if testUser.FolderPath != "" {
		folder = testUser.FolderPath + "/"
	}

	// Start multipart upload.
	m, err := json.Marshal(uploadFile{Key: folder + file.Name(), Size: int(fileInfo.Size()), RepositoryID: testUser.RepositoryID})
	if err != nil {
		t.Fatal("failed to open file:", err)
	}
	body := io.NopCloser(bytes.NewReader(m))
	header := http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	req := &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/file/upload-start"}, Proto: "2.0", Header: header, Body: body}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	req.AddCookie(testUser.Cookies[0])
	res, err := client.Do(req)
	if err != nil {
		t.Fatal("upload request failed:", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		t.Fatal("upload failed: status", res.Status)
	}
	uploadPartsRes := types.UploadStartResponse{}
	if err := json.NewDecoder(res.Body).Decode(&uploadPartsRes); err != nil {
		t.Fatal("Error decoding JSON:", err)
	}
	if len(uploadPartsRes.UploadParts) == 0 {
		t.Fatal("Server returned an empty user array")
	}

	// Resume the upload.
	m, err = json.Marshal(resumeFile{ID: uploadPartsRes.FileID})
	if err != nil {
		t.Fatal("failed to marshal:", err)
	}
	body = io.NopCloser(bytes.NewReader(m))
	header = http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	req = &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/file/upload-resume"}, Proto: "2.0", Header: header, Body: body}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	req.AddCookie(testUser.Cookies[0])
	res, err = client.Do(req)
	if err != nil {
		t.Fatal("upload request failed:", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		t.Fatal("upload failed: status", res.Status)
	}
	resumeParts := resumeFileResponse{}
	if err := json.NewDecoder(res.Body).Decode(&resumeParts); err != nil {
		t.Fatal("Error decoding JSON:", err)
	}
	if len(resumeParts.UploadParts) == 0 {
		t.Fatal("Server returned an empty user array")
	}

	partCount, partSize, leftover := fileutil.SplitFile(int(fileInfo.Size()))
	for i, part := range resumeParts.UploadParts {
		if i+1 == partCount && leftover != 0 {
			partSize = leftover
		}
		buffer := make([]byte, partSize)
		_, err = file.Read(buffer)
		if err != nil {
			t.Fatal("failed to read file:", err)
		}
		// Upload file part to s3.
		b := io.NopCloser(bytes.NewReader(buffer))
		awsReq, err := http.NewRequest("PUT", part.URL, b)
		if err != nil {
			t.Fatal("upload request failed:", err)
		}
		awsReq.ContentLength = int64(partSize)
		awsRes, err := client.Do(awsReq)
		if err != nil {
			t.Fatal("upload request failed:", err)
		}
		defer awsRes.Body.Close()
		if res.StatusCode >= 400 {
			t.Fatal("upload failed: status", res.Status)
		}
		etag := awsRes.Header.Get("ETag")

		// Post etag and part number to the server.
		reqPart := filePartRequest{FileID: uploadPartsRes.FileID}
		reqPart.ETag, reqPart.Part = etag, part.Part
		m, err = json.Marshal(reqPart)
		if err != nil {
			t.Fatal("failed to marshal:", err)
		}
		body = io.NopCloser(bytes.NewReader(m))
		header = http.Header{}
		header.Set("Content-Type", "application/json; charset=utf-8")
		req = &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/file/file-part"}, Proto: "2.0", Header: header, Body: body}
		if len(testUser.Cookies) == 0 {
			t.Fatal("Found no user's cookies to be sent")
		}
		req.AddCookie(testUser.Cookies[0])
		res, err = client.Do(req)
		if err != nil {
			t.Fatal("upload request failed:", err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			t.Fatal("upload failed: status", res.Status)
		}
	}

	m, err = json.Marshal(uploadCompleteRequest{ID: uploadPartsRes.FileID})
	if err != nil {
		t.Fatal("failed to marshal:", err)
	}
	body = io.NopCloser(bytes.NewReader(m))
	header = http.Header{}
	header.Set("Content-Type", "application/json; charset=utf-8")
	req = &http.Request{Method: "POST", URL: &url.URL{Scheme: "https", Host: serverHost, Path: "/api/file/upload-complete"}, Proto: "2.0", Header: header, Body: body}
	if len(testUser.Cookies) == 0 {
		t.Fatal("Found no user's cookies to be sent")
	}
	req.AddCookie(testUser.Cookies[0])
	res, err = client.Do(req)
	if err != nil {
		t.Fatal("upload request failed:", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		t.Fatal("upload failed: status", res.Status)
	}
}
