package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s Storage) DeleteFile(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.Bucket, Key: &key})
	if err != nil {
		return err
	}
	return nil
}
