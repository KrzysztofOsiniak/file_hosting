package storage

import (
	"backend/storage/aws"
	"backend/types"
)

// TODO: Make storage compatible with cloudflare's R2.

func InitStorage(storage string) {
	aws.InitStorage()
}

func StartUpload(key string, bytes int) (types.UploadStartResponse, error) {
	return aws.StartMultipartUpload(key, bytes)
}

func ResumeUpload() {
	// TODO
}

func CompleteUpload(key, uploadID string, completedParts []types.CompletePart) error {
	return aws.CompleteMultipartUpload(key, uploadID, completedParts)
}
