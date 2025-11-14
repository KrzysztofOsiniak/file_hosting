package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s Storage) GetDownload(ctx context.Context, key string) (string, error) {
	res, err := s.Presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Key: &key, Bucket: &s.Bucket, ResponseContentDisposition: aws.String(`attachment; filename="download"`),
	}, s3.WithPresignExpires(time.Minute))
	if err != nil {
		return "", err
	}
	return res.URL, nil
}
