package aws

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s Storage) GetDownload(ctx context.Context, key, name string) (string, error) {
	res, err := s.Presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Key: &key, Bucket: &s.Bucket, ResponseContentDisposition: aws.String(fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", "download", url.PathEscape(name))),
	}, s3.WithPresignExpires(time.Minute))
	if err != nil {
		return "", err
	}
	return res.URL, nil
}
