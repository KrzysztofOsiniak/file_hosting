package aws

import (
	"backend/types"
	"backend/util/fileutil"
	"context"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s Storage) ResumeMultipartUpload(ctx context.Context, key string, uploadID string, bytes int, completeParts []types.CompletePart) ([]types.UploadPart, error) {
	partCount, partSize, leftover := fileutil.SplitFile(bytes)
	uploads := []types.UploadPart{}
	var skipParts []int
	for _, v := range completeParts {
		skipParts = append(skipParts, v.Part)
	}
	for part := 1; part <= partCount; part++ {
		if slices.Contains(skipParts, part) {
			continue
		}
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
			return []types.UploadPart{}, err
		}
		uploads = append(uploads, types.UploadPart{URL: presignedPart.URL, Part: part})
	}
	return uploads, nil
}
