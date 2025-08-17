package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Delete all uploaded user's files.
// In progress multipart uploads should be deleted by lifecycle config.
func DeleteAllFiles(ctx context.Context, userID string) error {
	for {
		listInput := &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: aws.String(userID)}
		listOutput, err := client.ListObjectsV2(ctx, listInput)
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
			Bucket: aws.String(bucket),
			Delete: &s3types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		}
		_, err = client.DeleteObjects(ctx, delInput)
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
}
