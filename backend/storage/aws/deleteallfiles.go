package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/sync/errgroup"
)

// Delete all uploaded and in-progress files that contain a prefix.
// That prefix is expected to be a file path that ends with "/" for
// a precise match when deleting a user/repository or folder path.
func (s Storage) DeleteAllFiles(ctx context.Context, prefix string) error {
	group, groupCtx := errgroup.WithContext(ctx)

	// Get the last character in the string.
	lastChar := string([]rune(prefix)[len([]rune(prefix))-1])
	if lastChar != "/" {
		return errors.New("aws.DeleteAllFiles() error: prefix needs to end with a slash character (/)")
	}

	// Delete all fully uploaded objects.
	group.Go(func() error {
		for {
			listInput := &s3.ListObjectsV2Input{Bucket: &s.Bucket, Prefix: aws.String(prefix)}
			listOutput, err := s.Client.ListObjectsV2(groupCtx, listInput)
			if err != nil {
				return err
			}
			if len(listOutput.Contents) == 0 {
				return nil
			}

			var objects []s3types.ObjectIdentifier
			for _, obj := range listOutput.Contents {
				objects = append(objects, s3types.ObjectIdentifier{Key: obj.Key})
			}
			delInput := &s3.DeleteObjectsInput{
				Bucket: aws.String(s.Bucket),
				Delete: &s3types.Delete{
					Objects: objects,
					Quiet:   aws.Bool(true),
				},
			}
			_, err = s.Client.DeleteObjects(groupCtx, delInput)
			if err != nil {
				return err
			}

			// Continue if there are more objects.
			if listOutput.IsTruncated != nil && *listOutput.IsTruncated {
				listInput.ContinuationToken = listOutput.NextContinuationToken
			} else {
				return nil
			}
		}
	})

	// Delete all in-progress uploads.
	group.Go(func() error {
		paginator := s3.NewListMultipartUploadsPaginator(s.Client, &s3.ListMultipartUploadsInput{
			Bucket: &s.Bucket, Prefix: aws.String(prefix),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(groupCtx)
			if err != nil {
				return err
			}

			for _, upload := range page.Uploads {
				_, err := s.Client.AbortMultipartUpload(groupCtx, &s3.AbortMultipartUploadInput{
					Bucket:   &s.Bucket,
					Key:      upload.Key,
					UploadId: upload.UploadId,
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}
