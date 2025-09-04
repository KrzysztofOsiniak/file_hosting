package types

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
