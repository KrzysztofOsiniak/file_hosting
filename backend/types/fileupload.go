package types

import "time"

type UploadStart struct {
	UploadParts []UploadPart
	UploadID    string
	FileID      int
}

type UploadStartResponse struct {
	UploadParts []UploadPart `json:"uploadParts"`
	FileID      int          `json:"fileID"`
}

type UploadPart struct {
	URL  string `json:"url"`
	Part int    `json:"part"`
}

type CompletePart struct {
	ETag string
	Part int
}

type FileData struct {
	ID       int
	Date     *time.Time // Date is a pointer to check if it is null, meaning the file upload is not completed.
	UploadID string
}

type UploadedFile struct {
	ID string // Primary key of the file in the database as string, used in s3 as the object key.
}

type InProgressFile struct {
	ID       string // Primary key of the file in the database as string, used in s3 as the object key.
	UploadID string // Used when aborting an upload.
}
