package aws

import (
	"backend/types"
	"backend/util/fileutil"
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s Storage) StartMultipartUpload(ctx context.Context, key string, bytes int) (types.UploadStart, error) {
	init, err := s.Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return types.UploadStart{}, err
	}

	uploadID := *init.UploadId
	partCount, partSize, leftover := fileutil.SplitFile(bytes)
	uploads := []types.UploadPart{}
	for part := 1; part <= partCount; part++ {
		if part == partCount && leftover != 0 {
			partSize = leftover
		}
		presignedPart, err := s.Presigner.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket:        aws.String(s.Bucket),
			Key:           aws.String(key),
			UploadId:      aws.String(uploadID),
			PartNumber:    aws.Int32(int32(part)),
			ContentLength: aws.Int64(int64(partSize)),
		}, s3.WithPresignExpires(15*time.Minute))
		if err != nil {
			return types.UploadStart{}, err
		}
		uploads = append(uploads, types.UploadPart{URL: presignedPart.URL, Part: part})
	}

	return types.UploadStart{UploadParts: uploads, UploadID: uploadID}, nil
}
