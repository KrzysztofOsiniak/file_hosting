package seaweedfs

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Storage struct {
	Client    *s3.Client
	Presigner *s3.PresignClient
	Bucket    string
}

func InitStorage(sClient *s3.Client, sPresigner *s3.PresignClient, sBucket *string) {
	bucket := os.Getenv("LOCAL_BUCKET")
	if bucket == "" {
		log.Fatal("Loaded seaweedfs bucket from environment is not specified")
	}
	*sBucket = bucket
	key := os.Getenv("LOCAL_AWS_ACCESS_KEY_ID")
	if key == "" {
		log.Fatal("Loaded seaweedfs access key from environment is not specified")
	}
	secret := os.Getenv("LOCAL_AWS_SECRET_ACCESS_KEY")
	if secret == "" {
		log.Fatal("Loaded seaweedfs secret key from environment is not specified")
	}

	// LoadDefaultConfig reads AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID from
	// environment variables.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	creds := credentials.NewStaticCredentialsProvider(key, secret, "")
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-central-1"), config.WithCredentialsProvider(creds))
	if err != nil {
		log.Fatal("Failed to load configuration for seaweedfs: ", err)
	}
	*sClient = *s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://s3:8333/")
		o.Region = "eu-central-1"
		// Important to find docker host.
		o.UsePathStyle = true
	})
	*sPresigner = *s3.NewPresignClient(sClient)

	_, err = sClient.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: sBucket,
		ACL:    types.BucketCannedACLPrivate,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraintEuCentral1,
		},
	})
	// 409 code means the bucket already exists.
	if err != nil && strings.Contains(err.Error(), "409") {
		return
	}
	if err != nil {
		log.Fatal("Error creating a seaweedfs bucket: ", err)
	}
}
