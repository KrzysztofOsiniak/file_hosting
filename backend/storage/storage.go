package storage

import (
	"backend/storage/aws"
	"backend/storage/seaweedfs"
	"backend/types"
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type localStorage struct {
	AWS       aws.Storage
	Seaweedfs seaweedfs.Storage
}

type cloudStorage struct {
	AWS aws.Storage
}

var (
	ls = localStorage{AWS: aws.Storage{Client: &s3.Client{}, Presigner: &s3.PresignClient{}},
		Seaweedfs: seaweedfs.Storage{Client: &s3.Client{}, Presigner: &s3.PresignClient{}}}
	cs            = cloudStorage{AWS: aws.Storage{Client: &s3.Client{}, Presigner: &s3.PresignClient{}}}
	storageOption string
)

func InitStorage() {
	storageOption = os.Getenv("STORAGE_OPTION")
	if storageOption == "" {
		log.Fatal("Loaded STORAGE_OPTION from environment is not specified")
	}
	if storageOption != "cloud" && storageOption != "local" {
		log.Fatal("Loaded STORAGE_OPTION from environment is invalid")
	}
	if storageOption == "local" {
		seaweedfs.InitStorage(ls.Seaweedfs.Client, ls.Seaweedfs.Presigner, &ls.Seaweedfs.Bucket)
		seaweedfs.InitStorage(ls.AWS.Client, ls.AWS.Presigner, &ls.AWS.Bucket)
		return
	}
	if storageOption == "cloud" {
		aws.InitStorage(cs.AWS.Client, cs.AWS.Presigner, &cs.AWS.Bucket)
		return
	}
}

func StartUpload(ctx context.Context, key string, bytes int) (types.UploadStartResponse, error) {
	if storageOption == "local" {
		return ls.AWS.StartMultipartUpload(ctx, key, bytes)
	}
	if storageOption == "cloud" {
		return cs.AWS.StartMultipartUpload(ctx, key, bytes)
	}
	return types.UploadStartResponse{}, nil
}

func ResumeUpload(ctx context.Context, key string, uploadID string, bytes int, completeParts []types.CompletePart) ([]types.UploadPart, error) {
	if storageOption == "local" {
		return ls.AWS.ResumeMultipartUpload(ctx, key, uploadID, bytes, completeParts)
	}
	if storageOption == "cloud" {
		return cs.AWS.ResumeMultipartUpload(ctx, key, uploadID, bytes, completeParts)
	}
	return []types.UploadPart{}, nil
}

func CompleteUpload(ctx context.Context, key, uploadID string, completedParts []types.CompletePart) error {
	if storageOption == "local" {
		return ls.AWS.CompleteMultipartUpload(ctx, key, uploadID, completedParts)
	}
	if storageOption == "cloud" {
		return cs.AWS.CompleteMultipartUpload(ctx, key, uploadID, completedParts)
	}
	return nil
}

// Delete all files for a given user.
func DeleteAllFiles(ctx context.Context, userID string) error {
	if storageOption == "local" {
		return ls.AWS.DeleteAllFiles(ctx, userID)
	}
	if storageOption == "cloud" {
		return cs.AWS.DeleteAllFiles(ctx, userID)
	}
	return nil
}
