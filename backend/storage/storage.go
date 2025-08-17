package storage

import (
	"backend/storage/aws"
	"backend/types"
	"context"
)

// TODO: Make storage compatible with cloudflare's R2/self hosted aws s3 compatible storage.

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

// Delete all files for a given user.
func DeleteAllFiles(ctx context.Context, userID string) error {
	return aws.DeleteAllFiles(ctx, userID)
}
