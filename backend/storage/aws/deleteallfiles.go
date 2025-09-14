package aws

import (
	"backend/types"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/sync/errgroup"
)

// Delete all uploaded and in-progress files that are passed in arrays.
func (s Storage) DeleteAllFiles(ctx context.Context, uploadedFiles []types.UploadedFile, inProgressFiles []types.InProgressFile) error {
	group, groupCtx := errgroup.WithContext(ctx)

	// Delete all fully uploaded objects.
	group.Go(func() error {
		var objects []s3types.ObjectIdentifier
		filesCount := 0
		for _, file := range uploadedFiles {
			if filesCount == 1000 {
				filesCount = 0
				delInput := &s3.DeleteObjectsInput{
					Bucket: aws.String(s.Bucket),
					Delete: &s3types.Delete{
						Objects: objects,
						Quiet:   aws.Bool(true),
					},
				}
				_, err := s.Client.DeleteObjects(groupCtx, delInput)
				if err != nil {
					return err
				}
			}

			objects = append(objects, s3types.ObjectIdentifier{Key: &file.ID})
			filesCount += 1
		}

		if len(objects) == 0 {
			return nil
		}
		delInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(s.Bucket),
			Delete: &s3types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		}
		_, err := s.Client.DeleteObjects(groupCtx, delInput)
		if err != nil {
			return err
		}
		return nil
	})

	// Delete all in-progress uploads.
	group.Go(func() error {
		for _, file := range inProgressFiles {
			_, err := s.Client.AbortMultipartUpload(groupCtx, &s3.AbortMultipartUploadInput{
				Bucket:   &s.Bucket,
				Key:      &file.ID,
				UploadId: &file.UploadID,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}
	return nil
}
