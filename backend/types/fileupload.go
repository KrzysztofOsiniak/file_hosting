package types

type UploadStartResponse struct {
	UploadParts []UploadPart `json:"uploadParts"`
	UploadID    string       `json:"uploadID"`
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
