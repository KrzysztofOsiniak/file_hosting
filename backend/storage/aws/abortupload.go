package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s Storage) AbortUpload(ctx context.Context, key string, uploadID string) error {
	_, err := s.Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   &s.Bucket,
		Key:      &key,
		UploadId: &uploadID,
	})
	if err != nil {
		return err
	}
	return nil
}
