package aws

import (
	"backend/types"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (s Storage) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, completedParts []types.CompletePart) error {
	completed := []s3types.CompletedPart{}
	for _, v := range completedParts {
		completed = append(completed, s3types.CompletedPart{ETag: &v.ETag, PartNumber: aws.Int32(int32(v.Part))})
	}
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.Bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: completed,
		},
	}

	_, err := s.Client.CompleteMultipartUpload(ctx, completeInput)
	if err != nil {
		return err
	}
	return nil
}
